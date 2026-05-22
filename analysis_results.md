# IICPC Codebase — Latency Optimization Analysis

> **Baseline from logs/traces:** warm upload → engine ready ≈ 1.8–2.4s; cold/cache-miss spikes 5–19s; benchmark request p99 ≈ 385–551ms.

---

## Part 1: Your Optimisations.txt — Feasibility Verdicts

### 🟢 = Feasible & Recommended | 🟡 = Feasible with caveats | 🔴 = Not feasible or low-value

---

### Highest Impact (Items 1–5)

#### 1. Cache or skip repeated Docker builds
**Verdict: 🟢 Highly Feasible — BIGGEST WIN**

| Aspect | Detail |
|---|---|
| Code ref | [orchestrator_service.go:137](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/benchmark-orchestrator-go/internal/service/orchestrator_service.go#L137) |
| Current | `imageName := fmt.Sprintf("submission-%s:%d", shortID(submissionID), time.Now().UnixNano())` — always fresh tag |
| Impact | Directly attacks the 4–19s cold spikes |
| Difficulty | Medium |

The code generates a unique image name per build using `time.Now().UnixNano()` as a tag suffix. This means even identical submissions trigger a full Docker build. The submission service already computes a SHA-256 checksum in [submission_service.go:120](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/submission-service-go/internal/service/submission_service.go#L120) — you can use that as the image tag and skip `BuildImage()` entirely when the tag already exists locally.

**Expected reduction: 4–18s → ~200ms** for repeat submissions.

---

#### 2. Speed up container readiness detection
**Verdict: 🟢 Highly Feasible**

| Aspect | Detail |
|---|---|
| Code ref | [docker_manager.go:335](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/benchmark-orchestrator-go/internal/container/docker_manager.go#L335) |
| Current | Fixed 500ms ticker + `IsRunning()` Docker inspect on every probe |
| Impact | Could save 500ms–1.5s on every deployment |
| Difficulty | Low |

The `WaitForHealthy` method uses a constant 500ms ticker interval and performs _both_ `ContainerInspect` (via `IsRunning()`) and an HTTP health probe on every tick. For fast-starting containers, you waste half a second before the first check even happens.

**Recommended approach:**
- Start at 50ms, exponential backoff (50ms → 100ms → 200ms → 500ms)
- Skip `IsRunning()` if the HTTP probe succeeds (a 200 response proves it's running)
- Only call `IsRunning()` on HTTP errors to distinguish "still starting" from "crashed"

**Expected reduction: 500ms–1.5s** from deploy-to-ready path.

---

#### 3. Decouple Redpanda consumers from long Docker builds
**Verdict: 🟢 Feasible — Important for throughput**

| Aspect | Detail |
|---|---|
| Code ref | [orchestrator cmd/main.go:102–111](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/benchmark-orchestrator-go/cmd/main.go#L102-L111) |
| Current | `BuildAndDeploy(ctx, event.SubmissionID)` is called inline from the Kafka handler |
| Impact | Prevents consumer commit stalls under multiple concurrent submissions |
| Difficulty | Medium |

The `ValidationCompletedEvent` handler calls `BuildAndDeploy()` synchronously inside the consumer callback. If a Docker build takes 10s, the consumer group won't commit that offset for 10s, blocking all events on that partition.

**Fix:** Dispatch to a bounded worker pool (e.g., `chan struct{}` semaphore + goroutine), return `nil` immediately to commit the offset, and handle build results asynchronously.

---

#### 4. Fix bot fleet pacing
**Verdict: 🟢 Feasible — Improves measurement accuracy**

| Aspect | Detail |
|---|---|
| Code ref (worker) | [worker.go:34](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/bot-fleet-go/internal/bot/worker.go#L34) |
| Code ref (runner) | [runner.go:139](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/bot-fleet-go/internal/fleet/runner.go#L139) |
| Current | `time.NewTicker(cfg.InterRequestDelay)` per worker, plus `time.Sleep(100ms)` at runner end |
| Impact | Fixes under-driving OPS targets; improves measurement precision |
| Difficulty | Medium |

**Issues confirmed:**
1. The worker uses `time.NewTicker` but blocks on `client.Send()` inside the tick handler. If `Send()` takes 50ms and the interval is 100ms, effective throughput drops to 1/(50ms+100ms) instead of 1/100ms. A token-bucket or scheduled-dispatch approach would decouple pacing from request latency.
2. The 100ms `time.Sleep` in runner.go (line 139) is a hardcoded drain wait. A `sync.WaitGroup` + closing `resultsCh` is already used correctly, so this sleep adds 100ms of unnecessary tail latency to every benchmark run.
3. Each worker creates a new `http.Client` per call — no connection reuse. The `HTTPClient` struct in [http_client.go:36–43](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/bot-fleet-go/internal/bot/http_client.go#L36-L43) creates a fresh `http.Client` per worker which is fine for connection pooling, but doesn't configure `Transport` for connection reuse/keep-alive.

**Expected reduction:** More accurate OPS measurement; removes 100ms tail latency from each benchmark.

---

#### 5. Fix WebSocket hub blocking/deadlock risk
**Verdict: 🟢 Feasible — Critical correctness fix**

| Aspect | Detail |
|---|---|
| Code ref | [hub.go:79–93](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/websocket-service-go/internal/ws/hub.go#L79-L93) |
| Current | Sends to `h.unregister` channel while holding `RLock`, from inside the broadcast handler |
| Impact | Can deadlock or freeze all real-time updates |
| Difficulty | Low–Medium |

**The bug (confirmed in code):**
```go
// Line 88-90 — inside broadcast case, holding RLock
h.mu.RUnlock()         // unlock to allow unregister
h.unregister <- client // THIS CAN BLOCK if channel is full
h.mu.RLock()           // re-lock to continue loop
```

If the `unregister` channel is unbuffered (it is — [hub.go:37](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/websocket-service-go/internal/ws/hub.go#L37)), this can block indefinitely because the hub's own goroutine must drain `h.unregister`, but it's currently stuck trying to send to it.

**Fix:** Remove the client inline (under the write lock you already acquire) rather than sending through the channel. Alternatively, use `select` with `default` to make it non-blocking and defer cleanup.

---

### Backend / Infra (Items 6–12)

#### 6. Separate Redpanda producer profiles
**Verdict: 🟡 Feasible with caveats**

| Aspect | Detail |
|---|---|
| Code ref | [producer.go:24–29](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/pkg/events/producer.go#L24-L29) |
| Current | Single producer: 5ms linger, Snappy+LZ4 compression for ALL events |
| Impact | Could save 5ms per lifecycle event |
| Difficulty | Medium |

The `publish()` method uses `ProduceSync()` which waits for broker acknowledgement. The 5ms linger adds up for lifecycle events (engine_ready, benchmark_started, etc.) that are latency-sensitive. However, since telemetry snapshots already use `PublishAsync()` ([producer.go:108](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/pkg/events/producer.go#L108)), the impact is smaller than expected.

**Caveat:** Creating a second `kgo.Client` doubles connection overhead. A better approach: make lifecycle events use `PublishAsync` with a completion callback, and keep one client.

---

#### 7. Add DB pool settings
**Verdict: 🟢 Feasible — Easy win**

| Aspect | Detail |
|---|---|
| Code ref | [orchestrator cmd/main.go:48](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/benchmark-orchestrator-go/cmd/main.go#L48) |
| Current | `sqlx.Connect("postgres", cfg.Database.DSN())` with no pool config |
| Impact | Prevents connection churn under load |
| Difficulty | Very Low |

All Go services call `sqlx.Connect()` without tuning `SetMaxOpenConns`, `SetMaxIdleConns`, or `SetConnMaxLifetime`. Under concurrent benchmarks, this can cause connection churn and increased latency from TCP+TLS handshakes.

**Fix (3 lines per service):**
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
```

---

#### 8. Add benchmark_results(benchmark_id) index
**Verdict: 🟢 Feasible — Easy win**

| Aspect | Detail |
|---|---|
| Code ref | [orchestrator_repository.go:532–538](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/benchmark-orchestrator-go/internal/repository/orchestrator_repository.go#L532-L538) |
| Current | `WHERE benchmark_id = $1` queries on `benchmark_results` — no index on that column |
| Impact | Eliminates sequential scans as data grows |
| Difficulty | Very Low |

`GetBenchmarkResult` and `UpdateCorrectnessScore` both query `benchmark_results WHERE benchmark_id = $1`. The `ON CONFLICT (submission_id)` clause implies `submission_id` has a unique index, but `benchmark_id` does not. Same issue for `benchmark_history`.

**Fix:** `CREATE INDEX idx_benchmark_results_benchmark_id ON benchmark_results (benchmark_id);`

---

#### 9. Remove correctness race sleep
**Verdict: 🟢 Highly Feasible — Removes up to 5s latency**

| Aspect | Detail |
|---|---|
| Code ref | [orchestrator_service.go:606–612](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/benchmark-orchestrator-go/internal/service/orchestrator_service.go#L606-L612) |
| Current | Retry loop with `time.Sleep(1 * time.Second)` × 5 attempts |
| Impact | Removes up to 5s of blocking sleep in the consumer goroutine |
| Difficulty | Medium |

```go
for i := 0; i < 5; i++ {
    result, err = s.repo.GetBenchmarkResult(ctx, evt.BenchmarkID)
    if err == nil { break }
    time.Sleep(1 * time.Second)  // BLOCKS CONSUMER
}
```

This is a classic race: the correctness engine finishes before the orchestrator has created the `benchmark_results` row. Each sleep blocks the consumer goroutine for 1s.

**Fix:** Use `UpsertBenchmarkResult` to create a placeholder row when the benchmark starts (with zeroed metrics), so the correctness handler always finds a row. Or use a Postgres `INSERT ... ON CONFLICT` with a notification channel.

**Expected reduction: 0–5s** from correctness scoring path.

---

#### 10. Cache gateway health checks
**Verdict: 🟢 Feasible — Prevents cascading slowdowns**

| Aspect | Detail |
|---|---|
| Code ref | [api-gateway main.go:146–170](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/api-gateway-go/cmd/main.go#L146-L170) |
| Current | Every GET `/health` pings all 5 backends with HTTP |
| Impact | Prevents health checks from blocking under load |
| Difficulty | Low |

The health endpoint fires 5 concurrent HTTP requests on every call. If monitoring systems poll `/health` every second, that's 5 RPCs/second just for health. Already has a 1.2s timeout, so this is partially defended, but caching for 1–2s is easy.

---

#### 11. Reduce logging during benchmark
**Verdict: 🟡 Feasible — Moderate impact**

| Aspect | Detail |
|---|---|
| Code ref | [docker_manager.go:309–311](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/benchmark-orchestrator-go/internal/container/docker_manager.go#L309-L311) |
| Current | Every container stdout/stderr line logged at `Info` level |
| Impact | Reduces I/O overhead during benchmarks |
| Difficulty | Low |

```go
logger.Info(msg, slog.String("type", "container_runtime"), slog.String("stream", logType))
```

Every line of container stdout/stderr is logged at `Info`. For a chatty exchange engine producing 100+ lines/second, this adds measurable I/O. Change to `Debug` level and add rate-limiting.

---

#### 12. Switch Gin services to release mode
**Verdict: 🟢 Feasible — Trivial**

| Aspect | Detail |
|---|---|
| Code ref | [orchestrator cmd/main.go:152](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/benchmark-orchestrator-go/cmd/main.go#L152) |
| Current | `gin.Default()` — includes default logger + recovery middleware |
| Impact | Removes per-request debug logging overhead |
| Difficulty | Very Low |

The orchestrator uses `gin.Default()` which adds debug logging for every request. The API gateway correctly uses `gin.New()`. Fix: `gin.SetMode(gin.ReleaseMode)` + `gin.New()` + `gin.Recovery()`.

---

### Submission / Validation (Items 13–15)

#### 13. Keep streaming upload path
**Verdict: 🟢 Already good — No action needed**

The `validateAndStore()` function in [submission_service.go:198–232](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/submission-service-go/internal/service/submission_service.go#L198-L232) correctly uses `io.Pipe()` to stream validation + hashing + storage write concurrently. This is well-designed.

---

#### 14. Fix repository transaction executor bug
**Verdict: 🟢 Feasible — Correctness fix**

| Aspect | Detail |
|---|---|
| Code ref | [submission_repository.go:70–81](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/submission-service-go/internal/repository/submission_repository.go#L70-L81) |
| Current | `insertSubmission` accepts `exec submissionExecutor` but uses `r.db.ExecContext` |
| Impact | Correctness — bypasses transaction when called within `CreateWithNextVersion` |
| Difficulty | Very Low |

**Confirmed bug:** The `insertSubmission` method on line 81 uses `r.db.ExecContext` instead of `exec.ExecContext`, meaning when called from inside a transaction (line 56), the INSERT happens outside the transaction. This can cause data inconsistencies under concurrent submissions.

**Fix:** Change `r.db.ExecContext` → `exec.ExecContext` on line 81.

---

#### 15. Optimize validation extraction
**Verdict: 🟡 Feasible but lower priority**

The workspace analysis in `workspace.go` already does a single-pass walk. Optimizations like parallel extraction and buffer reuse would help for large archives (>50MB) but are lower priority given the measured baseline.

---

### C++ Engine Hot Path (Items 16–22)

#### 16. Replace `std::function` callbacks with interface/template
**Verdict: 🟡 Feasible — Medium gain, high effort**

| Aspect | Detail |
|---|---|
| Code ref | [MatchingEngine.h:28–30](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/trade-engine/include/MatchingEngine.h#L28-L30) |
| Current | `std::function<void(const Trade&)>` — heap-allocated type-erased wrapper |
| Impact | Removes indirect call overhead on every trade/execution |
| Difficulty | High (API redesign) |

`std::function` prevents inlining and involves a virtual dispatch + potential heap allocation. For a hot path executing per-trade, this matters. However, this is the **contestants' engine**, not the platform's code. If this is your reference implementation:
- **Feasible** to change to a CRTP sink or virtual interface
- **Expected impact:** 5–20ns per callback invocation, significant at high trade volumes

---

#### 17. Replace atomics with plain counters
**Verdict: 🟢 Feasible — Easy win**

| Aspect | Detail |
|---|---|
| Code ref | [MatchingEngine.h:115–116](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/trade-engine/include/MatchingEngine.h#L115-L116) |
| Current | `std::atomic<uint64_t> tradeIdCounter_{0}; std::atomic<uint64_t> currentTimestamp_{0};` |
| Impact | Removes unnecessary memory fence overhead |
| Difficulty | Very Low |

Since `MatchingEngine` is single-threaded (the `Exchange` wraps it with a `shared_mutex`), the atomics add unnecessary memory fences. Replace with plain `uint64_t`. The engine is explicitly documented as not thread-safe internally.

---

#### 18. Reuse scratch vectors per order
**Verdict: 🟢 Feasible — Good hot-path optimization**

| Aspect | Detail |
|---|---|
| Code ref | [MatchingEngine.cpp:82](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/trade-engine/src/MatchingEngine.cpp#L82), [MatchingEngine.cpp:400](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/trade-engine/src/MatchingEngine.cpp#L400) |
| Current | `std::vector<Trade> trades;` and `std::vector<Candidate> ordersToProcess;` allocated per order |
| Impact | Eliminates heap allocations on the critical matching path |
| Difficulty | Low |

Every `processLimitOrder` and `processMarketOrder` creates a fresh `std::vector<Trade>` (line 82, 164). Every `matchAtPriceLevel` creates a fresh `std::vector<Candidate>` (line 400). These should be member variables that are `.clear()`-ed per call to avoid repeated heap allocations.

**Expected reduction:** 1–5μs per order depending on allocation pattern.

---

#### 19. Replace `ExecutionResult.message` `std::string` with enum/const char*
**Verdict: 🟢 Feasible — Low-hanging fruit**

| Aspect | Detail |
|---|---|
| Code ref | [Order.h:187](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/trade-engine/include/Order.h#L187) |
| Current | `std::string message;` in `ExecutionResult` |
| Impact | Eliminates string allocation/copy on every order result |
| Difficulty | Low |

Every `ExecutionResult` allocates a `std::string` for human-readable messages like "Order added to book". There are ~15 unique messages. Use an enum + a `const char*` lookup table instead.

---

#### 20. Remove duplicate price-level lookups and system_clock calls
**Verdict: 🟢 Feasible — Measurable hot-path improvement**

| Aspect | Detail |
|---|---|
| Code ref | [MatchingEngine.cpp:589–604](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/trade-engine/src/MatchingEngine.cpp#L589-L604) |
| Current | `notifyBookMutation` calls `getQuantityAtPrice` + `getOrderCountAtPrice` (duplicate lookup) + `system_clock::now()` |
| Impact | Eliminates redundant map lookups and syscall per mutation |
| Difficulty | Low |

**Three issues confirmed:**
1. `getQuantityAtPrice` and `getOrderCountAtPrice` both perform independent `PriceLevel` lookups — could be a single lookup returning both values.
2. `system_clock::now()` is called per mutation — use the engine's monotonic timestamp counter instead.
3. `notifyBookMutation` is called after every fill _and_ after adding/removing resting orders — that's potentially N+1 mutation callbacks per order with N fills.

---

#### 21. Avoid global shared_mutex + string lookup in Exchange
**Verdict: 🟢 Feasible — Important for multi-symbol workloads**

| Aspect | Detail |
|---|---|
| Code ref | [Exchange.cpp:32–41](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/trade-engine/src/Exchange.cpp#L32-L41) |
| Current | `shared_lock(mutex_)` + `engines_.find(symbol)` on every `submitOrder` |
| Impact | Eliminates lock contention and string hashing on hot path |
| Difficulty | Medium |

Every order goes through a global `shared_mutex` lock + `std::unordered_map<std::string>` lookup. For single-symbol benchmarks this is just overhead. For multi-symbol, it serializes all symbols on the same lock.

**Fix:** Use integer symbol IDs (assigned at `createOrderBook` time) and per-engine queues. Or at minimum, `const auto it = engines_.find(symbol)` outside the lock using a reader pattern.

---

#### 22. Use C++20 + -O3 -march=native -flto
**Verdict: 🟢 Highly Feasible — Free performance**

| Aspect | Detail |
|---|---|
| Code ref | [CMakeLists.txt:9](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/trade-engine/CMakeLists.txt#L9), [CMakeLists.txt:25](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/trade-engine/CMakeLists.txt#L25) |
| Current | `CMAKE_CXX_STANDARD 17`, `-O2` in RelWithDebInfo |
| Impact | 10–30% throughput improvement for matching engine |
| Difficulty | Very Low |

```cmake
set(CMAKE_CXX_STANDARD 17)         # → 20
set(CMAKE_CXX_FLAGS_RELWITHDEBINFO "-O2 -g -DNDEBUG")  # → "-O3 -march=native -flto -g -DNDEBUG"
```

C++20 enables `std::span`, constexpr improvements, and better coroutine support. `-O3` enables loop vectorization, `-march=native` enables AVX/SSE, and `-flto` enables cross-TU inlining. These are free wins for compute-bound code.

> [!WARNING]
> `-march=native` makes binaries non-portable. If you build on one machine and deploy on another with a different CPU, use `-march=x86-64-v3` or similar instead.

---

## Part 2: Additional Optimizations Discovered

These are issues I found during code review that were **not** in your Optimisations.txt.

---

### A1. Bot HTTP client — No connection pooling/keep-alive
**Impact: Medium | Difficulty: Low**

| Aspect | Detail |
|---|---|
| Code ref | [http_client.go:36–43](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/bot-fleet-go/internal/bot/http_client.go#L36-L43) |
| Issue | Default `http.Transport` is used — no explicit keep-alive tuning |

Each bot worker creates a new `http.Client` per `NewHTTPClient`. While Go's default transport pools connections, it defaults to only 2 idle connections per host. For high-OPS benchmarks, this causes frequent TCP reconnections.

**Fix:** Share a single `http.Transport` with tuned `MaxIdleConnsPerHost`:
```go
transport := &http.Transport{
    MaxIdleConnsPerHost: 100,
    IdleConnTimeout:     90 * time.Second,
}
```

---

### A2. UUID generation on hot path
**Impact: Low–Medium | Difficulty: Low**

| Aspect | Detail |
|---|---|
| Code ref | [http_client.go:48](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/bot-fleet-go/internal/bot/http_client.go#L48) |
| Issue | `uuid.New().String()` per request uses `/dev/urandom` |

Every bot request calls `uuid.New()` which reads from the entropy pool. At 1000+ OPS, this adds ~1μs per call. Consider pre-generating UUIDs in a buffer or using a faster UUID-like ID generator (e.g., `xid` or atomic counter).

---

### A3. `json.Marshal` per order in bot Send()
**Impact: Low | Difficulty: Low**

| Aspect | Detail |
|---|---|
| Code ref | [http_client.go:57](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/bot-fleet-go/internal/bot/http_client.go#L57) |
| Issue | `json.Marshal(order)` allocates a new byte slice per request |

For high-OPS benchmarks, consider pre-serializing order templates or using `json.Encoder` with a pooled buffer.

---

### A4. ReadPump sends to unbuffered unregister channel
**Impact: Medium (correctness) | Difficulty: Low**

| Aspect | Detail |
|---|---|
| Code ref | [client.go:54](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/websocket-service-go/internal/ws/client.go#L54) |
| Issue | `c.hub.unregister <- c` in defer can block if hub is processing broadcast |

This is the same deadlock vector as item #5 but from the client side. The `ReadPump` defer sends to the unbuffered `unregister` channel, which can block if the hub is stuck in a broadcast loop.

---

### A5. API gateway log streaming uses polling loop
**Impact: Low | Difficulty: Low**

| Aspect | Detail |
|---|---|
| Code ref | [api-gateway main.go:112–125](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/api-gateway-go/cmd/main.go#L112-L125) |
| Issue | Busy-wait with `time.Sleep(500ms)` for new log lines |

```go
default:
    line, err := reader.ReadString('\n')
    if err != nil {
        time.Sleep(500 * time.Millisecond)
        continue
    }
```

This spins on the file descriptor. Use `fsnotify` or a `tail -f` approach instead.

---

### A6. Container log capture — per-line allocation
**Impact: Low | Difficulty: Low**

| Aspect | Detail |
|---|---|
| Code ref | [docker_manager.go:297](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/benchmark-orchestrator-go/internal/container/docker_manager.go#L297) |
| Issue | `make([]byte, size)` allocates a new buffer for every log line |

Use a `sync.Pool` or a pre-allocated ring buffer instead of allocating per line.

---

### A7. Leaderboard query joins submissions on every update
**Impact: Low–Medium | Difficulty: Medium**

| Aspect | Detail |
|---|---|
| Code ref | [orchestrator_repository.go:583–601](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/benchmark-orchestrator-go/internal/repository/orchestrator_repository.go#L583-L601) |
| Issue | Full leaderboard query (JOIN + ORDER BY + LIMIT) runs every time correctness or benchmark finishes |

`publishLeaderboardUpdated` is called from `ProcessBenchmarkFinished`, `ProcessCorrectnessEvaluated`, and `finishBenchmark`. Each call runs a full `SELECT ... JOIN submissions ... ORDER BY composite_score DESC` query. Consider caching the leaderboard in memory and only recomputing on score changes.

---

### A8. No prepared statements
**Impact: Low–Medium | Difficulty: Medium**

All repository methods build queries as string literals and pass them to `ExecContext`/`QueryRowContext`. PostgreSQL must parse and plan each query on every call. Using prepared statements (via `sqlx.Preparex`) or a statement cache would eliminate repeated planning overhead.

---

### A9. WebSocket broadcast channel size may be too small
**Impact: Medium | Difficulty: Very Low**

| Aspect | Detail |
|---|---|
| Code ref | [hub.go:35](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/websocket-service-go/internal/ws/hub.go#L35) |
| Current | `make(chan BroadcastMessage, 256)` |

With telemetry snapshots at 1/second per benchmark × N concurrent benchmarks × leaderboard updates, 256 may be tight. Consider increasing to 1024 or adding backpressure/coalescing for telemetry messages.

---

### A10. `ProcessBenchmarkFinished` approximates p50/p90 from p99
**Impact: Accuracy (not latency) | Difficulty: Low**

| Aspect | Detail |
|---|---|
| Code ref | [orchestrator_service.go:562–563](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/services/benchmark-orchestrator-go/internal/service/orchestrator_service.go#L562-L563) |

```go
P50LatencyMs: evt.P99LatencyMs * 0.5,  // approximate
P90LatencyMs: evt.P99LatencyMs * 0.8,
```

These are hardcoded multipliers. Pass real p50/p90 from the bot-fleet's `BenchmarkFinishedEvent` instead.

---

### A11. `canFillCompletely` does redundant full book scan for FOK
**Impact: Low | Difficulty: Medium**

| Aspect | Detail |
|---|---|
| Code ref | [MatchingEngine.cpp:519–570](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/trade-engine/src/MatchingEngine.cpp#L519-L570) |

For FOK orders, `canFillCompletely()` does a full scan of the book, then `matchOrder()` does the same scan again. Consider fusing these into a single pass.

---

### A12. `makeRejection` allocates std::string via `rejectReasonToString`
**Impact: Low | Difficulty: Very Low**

| Aspect | Detail |
|---|---|
| Code ref | [Order.h:196–203](file:///Users/chaitanyasaagar/Desktop/IICPC/IICPC/trade-engine/include/Order.h#L196-L203) |

`makeRejection` assigns `result.message = rejectReasonToString(reason)` which converts a `const char*` into a `std::string` allocation. If `message` stays as `std::string`, this is an avoidable copy.

---

## Part 3: Priority Matrix

| Priority | Item | Expected Latency Reduction | Effort |
|---|---|---|---|
| **P0** | #1 Cache Docker builds | 4–18s (cold path) | Medium |
| **P0** | #9 Remove correctness sleep | 0–5s | Medium |
| **P0** | #5 Fix WS hub deadlock | Prevents freezes | Low |
| **P0** | #14 Fix repo tx executor bug | Correctness | Very Low |
| **P1** | #2 Speed up health probing | 500ms–1.5s | Low |
| **P1** | #3 Decouple consumers from builds | Throughput | Medium |
| **P1** | #4 Fix bot fleet pacing | 100ms + accuracy | Medium |
| **P1** | #22 C++20 + O3/LTO | 10–30% engine perf | Very Low |
| **P1** | A1 Bot HTTP connection pooling | Variable | Low |
| **P2** | #7 DB pool settings | Variable | Very Low |
| **P2** | #8 Add benchmark_id index | Variable | Very Low |
| **P2** | #17 Plain counters | ~5ns/op | Very Low |
| **P2** | #18 Reuse scratch vectors | 1–5μs/order | Low |
| **P2** | #19 String → enum message | ~50ns/order | Low |
| **P2** | #12 Gin release mode | Minor | Very Low |
| **P3** | #6 Separate producer profiles | ~5ms/event | Medium |
| **P3** | #10 Cache health checks | Edge case | Low |
| **P3** | #11 Reduce logging | I/O savings | Low |
| **P3** | #16 Replace std::function | 5–20ns/call | High |
| **P3** | #20 Deduplicate lookups | ~10ns/mutation | Low |
| **P3** | #21 Symbol ID optimization | Multi-symbol only | Medium |

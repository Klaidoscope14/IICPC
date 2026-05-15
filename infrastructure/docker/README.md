# IICPC Local Database

The local Compose stack uses TimescaleDB on PostgreSQL 15. This gives the app normal relational tables for metadata and hypertables for high-write benchmark telemetry.

## Local Development

Run the full local stack from the repository root:

```bash
./scripts/dev-up.sh
```

If you already created a `postgres-data` volume before the switch to TimescaleDB, recreate the local database volume:

```bash
./scripts/dev-down.sh -v
./scripts/dev-up.sh
```

That deletes local database data. It is the right move for disposable dev data, not for production.

## Architecture

- `submissions`, `teams`, `validation_results`, `deployments`, and `benchmarks` store current platform metadata.
- `benchmark_results` stores the current best/latest result per submission for fast leaderboard reads.
- `benchmark_history` stores immutable completed benchmark runs.
- `telemetry_snapshots` is a TimescaleDB hypertable partitioned by `timestamp`.
- `leaderboard` is a live SQL view over indexed current result rows.

## Production Database

For production, use a managed PostgreSQL-compatible database with TimescaleDB support, such as Timescale Cloud or a self-managed TimescaleDB instance. Configure automated backups, point-in-time recovery, private networking, TLS, and separate application credentials with least privilege.

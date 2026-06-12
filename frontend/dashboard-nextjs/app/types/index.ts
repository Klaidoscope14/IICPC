// LeaderboardEntry represents a single row in the leaderboard.
export interface LeaderboardEntry {
  rank: number
  submissionId?: string
  benchmarkId?: string
  team: string
  tps: number
  latency: number
  correctness: number
  score: number
}

// BackendLeaderboardEntry is the API gateway/orchestrator leaderboard contract.
export interface BackendLeaderboardEntry {
  rank: number
  submission_id?: string
  benchmark_id?: string
  team_name: string
  tps: number
  p50_latency_ms: number
  p90_latency_ms: number
  p99_latency_ms: number
  correctness_score: number
  total_orders: number
  failed_orders: number
  composite_score: number
}

// Submission represents a contestant's code upload and its status.
export interface Submission {
  id: string
  contestant_id: string
  team_name: string
  language: string
  status: string
  created_at: string
  updated_at: string
  version?: number
  benchmark_results?: BenchmarkResult[]
}

// BenchmarkResult holds performance metrics from a benchmark run.
export interface BenchmarkResult {
  metrics?: {
    total_orders_sent?: number
    avg_latency_ms?: number
    correctness?: number
  }
  score?: number
}

// SubmissionFormData holds the form state for the submission form.
export interface SubmissionFormData {
  teamName: string
  language: string
  dockerfile: string
  benchmarkPreset?: string
}

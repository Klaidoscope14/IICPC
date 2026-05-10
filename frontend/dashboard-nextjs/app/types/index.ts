// LeaderboardEntry represents a single row in the leaderboard.
export interface LeaderboardEntry {
  rank: number
  team: string
  tps: number
  latency: number
  correctness: number
  score: number
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
  contestantId: string
  teamName: string
  language: string
  dockerfile: string
}

import { CheckCircle, Clock, PlayCircle, AlertTriangle } from 'lucide-react'

export type SubmissionStatus = 'pending' | 'uploaded' | 'validation_queued' | 'validating' | 'validated' | 'validation_failed' | 'processing' | 'deploying' | 'deployed' | 'benchmarking' | 'completed' | 'failed'

export function SubmissionStatusBadge({ status }: { status: SubmissionStatus | string }) {
  const formatStatus = (s: string) => s.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase())

  const getStatusConfig = (s: string) => {
    switch (s) {
      case 'completed':
      case 'validated':
        return { color: 'bg-green-600/20 text-green-400 border-green-500/30', icon: <CheckCircle className="h-3 w-3" /> }
      case 'validation_failed':
      case 'failed':
        return { color: 'bg-red-600/20 text-red-400 border-red-500/30', icon: <AlertTriangle className="h-3 w-3" /> }
      case 'benchmarking':
      case 'validating':
      case 'deploying':
      case 'processing':
        return { color: 'bg-blue-600/20 text-blue-400 border-blue-500/30', icon: <PlayCircle className="h-3 w-3 animate-pulse" /> }
      default:
        return { color: 'bg-yellow-600/20 text-yellow-400 border-yellow-500/30', icon: <Clock className="h-3 w-3" /> }
    }
  }

  const { color, icon } = getStatusConfig(status)

  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${color}`}>
      {icon}
      {formatStatus(status)}
    </span>
  )
}

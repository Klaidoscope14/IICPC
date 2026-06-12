import { BenchmarkMonitor } from '../../components/BenchmarkMonitor'

export const metadata = {
  title: 'Live Monitor | IICPC',
}

export default function MonitorPage() {
  return (
    <div className="py-8">
      <BenchmarkMonitor />
    </div>
  )
}

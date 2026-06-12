import { LogsViewer } from '../../../components/LogsViewer'
import { ArrowLeft } from 'lucide-react'
import Link from 'next/link'

export const metadata = {
  title: 'Submission Details | IICPC',
}

export default function SubmissionDetailsPage({ params }: { params: { id: string } }) {
  return (
    <div className="max-w-5xl mx-auto space-y-6">
      <div className="flex items-center gap-4">
        <Link href="/benchmarks" className="p-2 hover:bg-slate-800 rounded-full transition-colors">
          <ArrowLeft className="w-5 h-5 text-slate-400" />
        </Link>
        <div>
          <h2 className="text-2xl font-bold text-white flex items-center gap-3">
            Submission <span className="font-mono text-lg text-blue-400">{params.id}</span>
          </h2>
          <p className="text-slate-400 mt-1">Detailed logs and execution status</p>
        </div>
      </div>

      <LogsViewer submissionId={params.id} />
    </div>
  )
}

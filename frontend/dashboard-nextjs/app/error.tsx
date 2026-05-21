'use client'

import { useEffect } from 'react'
import { AlertCircle, RefreshCcw } from 'lucide-react'

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  useEffect(() => {
    // Log the error to an error reporting service
    console.error(error)
  }, [error])

  return (
    <div className="min-h-[50vh] flex flex-col items-center justify-center p-8 text-center">
      <div className="bg-red-500/10 p-4 rounded-full mb-6 border border-red-500/20">
        <AlertCircle className="w-12 h-12 text-red-500" />
      </div>
      <h2 className="text-2xl font-semibold text-slate-100 mb-2">Something went wrong!</h2>
      <p className="text-slate-400 mb-8 max-w-md">
        We encountered an unexpected error while rendering this page. Our systems have logged the issue.
      </p>
      <button
        onClick={() => reset()}
        className="flex items-center gap-2 px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-slate-900"
      >
        <RefreshCcw className="w-4 h-4" />
        Try again
      </button>
    </div>
  )
}

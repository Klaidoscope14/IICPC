export default function Loading() {
  return (
    <div className="space-y-6 animate-pulse p-4">
      <div className="h-8 bg-slate-800 rounded w-1/4 mb-8"></div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-32 bg-slate-800 rounded-xl border border-slate-700/50"></div>
        ))}
      </div>

      <div className="h-64 bg-slate-800 rounded-xl border border-slate-700/50 mt-8"></div>
    </div>
  )
}

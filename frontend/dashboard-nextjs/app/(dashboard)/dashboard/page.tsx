'use client'

import { useEffect, useState } from 'react'
import { Activity, Server, Database, Network, Shield, Cpu, AlertCircle, CheckCircle2, XCircle } from 'lucide-react'

interface HealthResponse {
  status: string
  service: string
  version: string
  checks: Record<string, string>
  timestamp: string
}

type ServiceStatus = 'operational' | 'degraded' | 'down' | 'loading'

/**
 * Home Page - System Status Dashboard
 * Fetches real health data from the API gateway.
 */
export default function Home() {
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchHealth = async () => {
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8082'
      const res = await fetch(`${apiUrl}/health`, { cache: 'no-store' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data: HealthResponse = await res.json()
      setHealth(data)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reach API gateway')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchHealth()
    const interval = setInterval(fetchHealth, 10000)
    return () => clearInterval(interval)
  }, [])

  const getCheckStatus = (checkName: string): ServiceStatus => {
    if (loading) return 'loading'
    if (!health) return 'down'
    const check = health.checks[checkName]
    return check === 'ok' ? 'operational' : 'down'
  }

  const gatewayStatus: ServiceStatus = loading ? 'loading' : health?.status === 'healthy' ? 'operational' : 'down'

  return (
    <div className="max-w-6xl mx-auto space-y-8 animate-in fade-in duration-700">
      <div className="flex items-center justify-between mb-10">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-blue-500/10 rounded-xl border border-blue-500/20">
            <Activity className="h-8 w-8 text-blue-400 drop-shadow-[0_0_8px_rgba(96,165,250,0.8)]" />
          </div>
          <div>
            <h2 className="text-3xl font-bold text-white tracking-tight">System Status</h2>
            <p className="text-slate-400 mt-1 flex items-center gap-2">
              Real-time infrastructure health
              {health && (
                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-800 text-slate-300 border border-slate-700">
                  Updated: {new Date(health.timestamp).toLocaleTimeString()}
                </span>
              )}
            </p>
          </div>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-3 px-5 py-4 rounded-xl bg-red-950/40 text-red-400 border border-red-500/30 text-sm backdrop-blur-md shadow-lg shadow-red-900/20">
          <AlertCircle className="w-5 h-5 shrink-0" />
          <span className="font-medium">Gateway Error: {error}</span>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <StatusCard 
          title="API Gateway" 
          icon={<Network className="w-6 h-6 text-indigo-400" />}
          status={gatewayStatus}
          metrics={[
            { label: 'Version', value: health?.version || '--' },
            { label: 'Status', value: health?.status || '--' },
          ]}
        />
        <StatusCard 
          title="Submission Service" 
          icon={<Server className="w-6 h-6 text-cyan-400" />}
          status={getCheckStatus('submission-service')}
          metrics={[
            { label: 'Health', value: health?.checks['submission-service'] || '--' },
          ]}
        />
        <StatusCard 
          title="Validation Service" 
          icon={<Shield className="w-6 h-6 text-emerald-400" />}
          status={getCheckStatus('validation-service')}
          metrics={[
            { label: 'Health', value: health?.checks['validation-service'] || '--' },
          ]}
        />
        <StatusCard 
          title="Benchmark Orchestrator" 
          icon={<Cpu className="w-6 h-6 text-purple-400" />}
          status={getCheckStatus('benchmark-orchestrator')}
          metrics={[
            { label: 'Health', value: health?.checks['benchmark-orchestrator'] || '--' },
          ]}
        />
        <StatusCard 
          title="Bot Fleet" 
          icon={<Activity className="w-6 h-6 text-amber-400" />}
          status={getCheckStatus('bot-fleet')}
          metrics={[
            { label: 'Health', value: health?.checks['bot-fleet'] || '--' },
          ]}
        />
        <StatusCard 
          title="Redpanda Message Bus" 
          icon={<Database className="w-6 h-6 text-rose-400" />}
          status={gatewayStatus}
          metrics={[
            { label: 'Broker', value: gatewayStatus === 'operational' ? 'Connected' : '--' },
          ]}
        />
      </div>
    </div>
  )
}

function StatusCard({ title, icon, status, metrics }: { title: string, icon: React.ReactNode, status: ServiceStatus, metrics: {label: string, value: string}[] }) {
  const statusConfig = {
    operational: {
      color: 'text-emerald-400',
      bg: 'bg-emerald-500/10',
      border: 'border-emerald-500/20',
      dot: 'bg-emerald-500',
      label: 'OPERATIONAL'
    },
    degraded: {
      color: 'text-amber-400',
      bg: 'bg-amber-500/10',
      border: 'border-amber-500/20',
      dot: 'bg-amber-500',
      label: 'DEGRADED'
    },
    down: {
      color: 'text-rose-400',
      bg: 'bg-rose-500/10',
      border: 'border-rose-500/20',
      dot: 'bg-rose-500',
      label: 'DOWN'
    },
    loading: {
      color: 'text-slate-400',
      bg: 'bg-slate-500/10',
      border: 'border-slate-500/20',
      dot: 'bg-slate-500',
      label: 'CHECKING...'
    },
  }

  const conf = statusConfig[status]

  return (
    <div className="glass-panel glass-panel-hover rounded-2xl p-6 group relative overflow-hidden">
      <div className={`absolute top-0 left-0 w-1 h-full ${conf.bg} border-l-2 ${conf.border}`}></div>
      <div className="flex items-start justify-between mb-6">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-slate-800/50 border border-slate-700 group-hover:scale-110 transition-transform duration-300">
            {icon}
          </div>
          <h3 className="text-slate-200 font-semibold tracking-wide">{title}</h3>
        </div>
      </div>
      
      <div className="space-y-4 mb-6">
        {metrics.map((m, i) => (
          <div key={i} className="flex justify-between items-center text-sm border-b border-slate-800 pb-2 last:border-0 last:pb-0">
            <span className="text-slate-400">{m.label}</span>
            <span className="text-slate-200 font-mono bg-slate-900/50 px-2 py-0.5 rounded">{m.value}</span>
          </div>
        ))}
      </div>

      <div className="pt-4 border-t border-slate-800/60 flex items-center justify-between">
        <span className="text-xs text-slate-500 font-medium">STATUS</span>
        <div className={`flex items-center gap-2 px-3 py-1 rounded-full text-xs font-bold tracking-wider ${conf.bg} ${conf.color} border ${conf.border}`}>
          <div className="relative flex h-2 w-2">
            {status === 'operational' && (
              <span className={`animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 ${conf.dot}`}></span>
            )}
            <span className={`relative inline-flex rounded-full h-2 w-2 ${conf.dot}`}></span>
          </div>
          {conf.label}
        </div>
      </div>
    </div>
  )
}

import { ReactNode } from 'react'

interface EmptyStateProps {
  icon: ReactNode
  title: string
  description?: string
  action?: ReactNode
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="text-center py-16 px-4">
      <div className="flex justify-center mb-6 opacity-40">
        {icon}
      </div>
      <h3 className="text-xl font-medium text-slate-200 mb-2">{title}</h3>
      {description && <p className="text-slate-400 max-w-md mx-auto mb-6">{description}</p>}
      {action && <div>{action}</div>}
    </div>
  )
}

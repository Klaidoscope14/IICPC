'use client'

type Tab = 'submit' | 'leaderboard' | 'submissions' | 'monitor'

interface TabNavigationProps {
  activeTab: Tab
  onTabChange: (tab: Tab) => void
}

const TABS: { key: Tab; label: string }[] = [
  { key: 'submit', label: 'Submit Code' },
  { key: 'leaderboard', label: 'Leaderboard' },
  { key: 'submissions', label: 'My Submissions' },
  { key: 'monitor', label: '⚡ Live Monitor' },
]

/**
 * Horizontal tab navigation bar.
 */
export function TabNavigation({ activeTab, onTabChange }: TabNavigationProps) {
  return (
    <div className="border-b border-slate-700 bg-slate-900/30">
      <div className="container mx-auto px-4">
        <div className="flex gap-1">
          {TABS.map((tab) => (
            <button
              key={tab.key}
              onClick={() => onTabChange(tab.key)}
              className={`px-6 py-3 text-sm font-medium transition-colors ${
                activeTab === tab.key
                  ? 'text-blue-400 border-b-2 border-blue-400'
                  : 'text-slate-400 hover:text-slate-300'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

export type { Tab }

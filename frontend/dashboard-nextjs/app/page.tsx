'use client'

import { useState } from 'react'
import { Header } from './components/Header'
import { TabNavigation, type Tab } from './components/TabNavigation'
import { SubmissionForm } from './components/SubmissionForm'
import { Leaderboard } from './components/Leaderboard'
import { SubmissionsList } from './components/SubmissionsList'
import { BenchmarkMonitor } from './components/BenchmarkMonitor'

/**
 * Root page — composes the header, tab navigation, and active tab content.
 */
export default function Home() {
  const [activeTab, setActiveTab] = useState<Tab>('submit')

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900">
      <Header />
      <TabNavigation activeTab={activeTab} onTabChange={setActiveTab} />

      <main className="container mx-auto px-4 py-8">
        {activeTab === 'submit' && <SubmissionForm />}
        {activeTab === 'leaderboard' && <Leaderboard />}
        {activeTab === 'submissions' && <SubmissionsList />}
        {activeTab === 'monitor' && <BenchmarkMonitor />}
      </main>
    </div>
  )
}

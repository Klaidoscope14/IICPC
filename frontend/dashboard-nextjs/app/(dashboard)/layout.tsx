import '../globals.css'
import { Header } from '../components/Header'
import { Navbar } from '../components/Navbar'

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="min-h-screen bg-[#09090b] text-slate-50 selection:bg-blue-500/30 font-sans">
      <div className="min-h-screen bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-slate-900 via-[#09090b] to-[#09090b]">
        <div className="sticky top-0 z-50 w-full border-b border-slate-800/60 bg-slate-950/50 backdrop-blur-md">
          <Header />
          <Navbar />
        </div>
        <main className="container mx-auto px-4 py-8 relative z-10">
          {children}
        </main>
      </div>
    </div>
  )
}

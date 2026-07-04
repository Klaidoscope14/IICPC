'use client'

import { useState, useEffect } from 'react'
import { Zap, Timer, LogOut, User } from 'lucide-react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'

/**
 * Application header bar with branding and contest timer.
 */
export function Header() {
  const router = useRouter();
  const [timeLeft, setTimeLeft] = useState<string>('00:00:00');
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [teamName, setTeamName] = useState<string | null>(null);

  useEffect(() => {
    const hasToken = document.cookie.split('; ').some(c => c.startsWith('token='));
    setIsAuthenticated(hasToken);

    if (hasToken) {
      const token = document.cookie.split('; ').find(c => c.startsWith('token='))?.split('=')[1];
      fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8082'}/api/auth/profile`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
        .then(res => res.json())
        .then(data => {
          if (data && data.team && data.team.team_name) {
            setTeamName(data.team.team_name);
          }
        })
        .catch(console.error);
    }

    fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8082'}/api/v1/admin/hackathon/dates`)
      .then(res => res.json())
      .then(data => {
        if (data && data.end_date) {
          const target = new Date(data.end_date).getTime();
          const timer = setInterval(() => {
            const diff = target - Date.now();
            if (diff > 0) {
              const h = Math.floor(diff / (1000 * 60 * 60));
              const m = Math.floor((diff / 1000 / 60) % 60);
              const s = Math.floor((diff / 1000) % 60);
              setTimeLeft(`${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`);
            } else {
              setTimeLeft('00:00:00');
              clearInterval(timer);
            }
          }, 1000);
          return () => clearInterval(timer);
        }
      })
      .catch(console.error);
  }, []);

  const handleLogout = () => {
    document.cookie = 'token=; path=/; max-age=0; SameSite=Lax';
    setIsAuthenticated(false);
    router.push('/');
  };

  return (
    <header className="border-b-0 bg-transparent">
      <div className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3 group cursor-pointer">
            <div className="relative">
              <div className="absolute inset-0 bg-blue-500 blur-md opacity-50 group-hover:opacity-100 transition-opacity duration-500 rounded-full"></div>
              <Zap className="relative h-8 w-8 text-blue-400 drop-shadow-[0_0_8px_rgba(96,165,250,0.8)]" />
            </div>
            <h1 className="text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 via-indigo-400 to-purple-400 tracking-tight text-glow">
              <Link href="/">IICPC Dashboard</Link>
            </h1>
          </div>
          <div className="flex items-center gap-6">
            <div className="hidden md:flex items-center gap-2 px-4 py-1.5 rounded-full bg-slate-800/50 border border-slate-700/50 shadow-inner">
              <Timer className="w-4 h-4 text-emerald-400" />
              <span className="text-emerald-400 font-mono text-sm font-medium tracking-wider">{timeLeft}</span>
              <span className="text-slate-500 text-xs ml-1">REMAINING</span>
            </div>
            {isAuthenticated ? (
              <div className="flex items-center gap-4">
                {teamName && (
                  <span className="text-sm font-medium text-emerald-400">
                    Welcome, {teamName}
                  </span>
                )}
                <button onClick={handleLogout} className="flex items-center gap-2 text-sm text-slate-300 hover:text-white transition">
                  <LogOut className="w-4 h-4" /> Logout
                </button>
              </div>
            ) : (
              <Link href="/auth/login" className="flex items-center gap-2 text-sm text-blue-400 hover:text-blue-300 transition">
                <User className="w-4 h-4" /> Login
              </Link>
            )}
          </div>
        </div>
      </div>
    </header>
  )
}


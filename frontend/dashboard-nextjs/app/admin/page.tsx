'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function AdminDashboard() {
  const router = useRouter();
  const [password, setPassword] = useState('');
  const [token, setToken] = useState<string | null>(null);
  const [stats, setStats] = useState<any>(null);
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [message, setMessage] = useState('');

  useEffect(() => {
    const t = localStorage.getItem('admin_token');
    if (t) {
      setToken(t);
      fetchStats(t);
    }
  }, []);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || ''}/api/v1/admin/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      });
      if (res.ok) {
        const data = await res.json();
        setToken(data.token);
        localStorage.setItem('admin_token', data.token);
        fetchStats(data.token);
      } else {
        alert('Invalid admin credentials');
      }
    } catch (err) {
      console.error(err);
    }
  };

  const fetchStats = async (t: string) => {
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || ''}/api/v1/admin/stats`, {
        headers: { Authorization: `Bearer ${t}` },
      });
      if (res.ok) {
        const data = await res.json();
        setStats(data);
      } else if (res.status === 401) {
        setToken(null);
        localStorage.removeItem('admin_token');
      }
    } catch (err) {
      console.error(err);
    }
  };

  const updateDates = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || ''}/api/v1/admin/hackathon/dates`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          start_date: new Date(startDate).toISOString(),
          end_date: new Date(endDate).toISOString(),
        }),
      });
      if (res.ok) {
        setMessage('Hackathon dates updated successfully!');
      } else {
        setMessage('Failed to update dates');
      }
    } catch (err) {
      console.error(err);
    }
  };

  if (!token) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-950 p-4">
        <form onSubmit={handleLogin} className="w-full max-w-md space-y-6 rounded-2xl bg-gray-900 p-8 shadow-xl border border-gray-800">
          <h2 className="text-2xl font-bold text-white text-center">Admin Login</h2>
          <input
            type="password"
            placeholder="Admin Password"
            className="w-full rounded-md bg-gray-800 border-gray-700 text-white p-3"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <button type="submit" className="w-full rounded-md bg-red-600 p-3 font-semibold text-white hover:bg-red-500">
            Login
          </button>
        </form>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="max-w-6xl mx-auto space-y-8">
        <div className="flex justify-between items-center">
          <h1 className="text-3xl font-bold">Admin Dashboard</h1>
          <button 
            onClick={() => { setToken(null); localStorage.removeItem('admin_token'); }}
            className="bg-gray-800 px-4 py-2 rounded text-sm hover:bg-gray-700"
          >
            Logout
          </button>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="bg-gray-900 p-6 rounded-xl border border-gray-800">
            <h3 className="text-gray-400 text-sm uppercase">Total Users</h3>
            <p className="text-4xl font-bold text-blue-400 mt-2">{stats?.total_users || 0}</p>
          </div>
          <div className="bg-gray-900 p-6 rounded-xl border border-gray-800">
            <h3 className="text-gray-400 text-sm uppercase">Total Teams</h3>
            <p className="text-4xl font-bold text-green-400 mt-2">{stats?.total_teams || 0}</p>
          </div>
          <div className="bg-gray-900 p-6 rounded-xl border border-gray-800">
            <h3 className="text-gray-400 text-sm uppercase">Total Submissions</h3>
            <p className="text-4xl font-bold text-purple-400 mt-2">{stats?.total_submissions || 0}</p>
          </div>
        </div>

        <div className="bg-gray-900 p-6 rounded-xl border border-gray-800">
          <h2 className="text-xl font-bold mb-4">Configure Hackathon Dates</h2>
          {message && <p className="mb-4 text-green-400">{message}</p>}
          <form onSubmit={updateDates} className="grid grid-cols-1 md:grid-cols-3 gap-4 items-end">
            <div>
              <label className="block text-sm text-gray-400 mb-1">Start Date</label>
              <input 
                type="datetime-local" 
                className="w-full bg-gray-800 border-gray-700 rounded p-2 text-white"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                required
              />
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1">End Date</label>
              <input 
                type="datetime-local" 
                className="w-full bg-gray-800 border-gray-700 rounded p-2 text-white"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                required
              />
            </div>
            <button type="submit" className="bg-blue-600 hover:bg-blue-500 text-white font-semibold p-2 rounded">
              Update Dates
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}

'use client'

import { useState } from 'react'
import { Upload } from 'lucide-react'
import { apiClient } from '../lib/api'
import type { SubmissionFormData } from '../types'

const ALLOWED_EXTENSIONS = ['.zip', '.tar', '.tar.gz']
const ALLOWED_MIME_TYPES = [
  'application/zip',
  'application/x-zip-compressed',
  'application/x-tar',
  'application/gzip',
]

/**
 * Form for uploading a trading engine submission.
 * Supports drag-and-drop and click-to-browse file upload.
 */
export function SubmissionForm() {
  const [formData, setFormData] = useState<SubmissionFormData>({
    teamName: '',
    language: 'cpp',
    dockerfile: '',
    benchmarkPreset: 'medium_traffic',
  })
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [isDragging, setIsDragging] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  const isAllowedFile = (file: File): boolean => {
    return (
      ALLOWED_MIME_TYPES.includes(file.type) ||
      ALLOWED_EXTENSIONS.some((ext) => file.name.endsWith(ext))
    )
  }

  const handleFileSelect = (file: File) => {
    if (isAllowedFile(file)) {
      setSelectedFile(file)
      setMessage(null)
    } else {
      setMessage({ type: 'error', text: 'Please upload a ZIP or TAR.GZ file.' })
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)
    if (e.dataTransfer.files?.length > 0) {
      handleFileSelect(e.dataTransfer.files[0])
      e.dataTransfer.clearData()
    }
  }

  const handleDragEnter = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(true)
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(true)
  }

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)
  }

  const handleFileInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) handleFileSelect(file)
  }

  const updateField = (field: keyof SubmissionFormData, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setMessage(null)

    if (!selectedFile) {
      setMessage({ type: 'error', text: 'Please select a code archive file.' })
      return
    }

    setSubmitting(true)

    try {
      const form = new FormData()
      form.append('team_name', formData.teamName)
      form.append('language', formData.language)
      form.append('dockerfile', formData.dockerfile || '')
      form.append('benchmark_preset', formData.benchmarkPreset || 'medium_traffic')
      form.append('code_archive', selectedFile)

      const result = await apiClient<{ id: string }>('/api/v1/submissions', {
        method: 'POST',
        body: form,
      })

      setMessage({ type: 'success', text: `Submission successful! ID: ${result.id}` })

      // Persist identity for leaderboard highlighting.
      localStorage.setItem('iicpc_team_name', formData.teamName)

      // Reset form.
      setFormData({ teamName: '', language: 'cpp', dockerfile: '', benchmarkPreset: 'medium_traffic' })
      setSelectedFile(null)
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : 'Error connecting to submission service.'
      setMessage({ type: 'error', text: errorMessage })
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="max-w-2xl mx-auto animate-in fade-in slide-in-from-bottom-8 duration-700">
      <div className="glass-panel rounded-2xl p-8 relative overflow-hidden">
        {/* Subtle decorative glow */}
        <div className="absolute top-0 right-0 w-64 h-64 bg-blue-500/10 rounded-full blur-[80px] pointer-events-none"></div>

        <div className="flex items-center gap-4 mb-8">
          <div className="p-3 bg-blue-500/10 rounded-xl border border-blue-500/20">
            <Upload className="h-6 w-6 text-blue-400 drop-shadow-[0_0_8px_rgba(96,165,250,0.8)]" />
          </div>
          <h2 className="text-2xl font-bold text-white tracking-tight">Submit Your Trading Engine</h2>
        </div>

        {/* Inline status message */}
        {message && (
          <div
            className={`mb-8 px-5 py-4 rounded-xl text-sm font-medium backdrop-blur-md shadow-lg transition-all ${
              message.type === 'success'
                ? 'bg-emerald-950/40 text-emerald-400 border border-emerald-500/30 shadow-emerald-900/20'
                : 'bg-red-950/40 text-red-400 border border-red-500/30 shadow-red-900/20'
            }`}
          >
            {message.text}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6 relative z-10">
          <div className="group">
            <label className="block text-sm font-medium text-slate-300 mb-2 transition-colors group-focus-within:text-blue-400">Team Name</label>
            <input
              id="team-name"
              type="text"
              required
              value={formData.teamName}
              onChange={(e) => updateField('teamName', e.target.value)}
              className="w-full px-4 py-3 bg-slate-900/50 border border-slate-700/50 rounded-xl text-white placeholder-slate-500 transition-all duration-300 focus:outline-none focus:border-blue-500/50 focus:ring-1 focus:ring-blue-500/50 focus:bg-slate-900 shadow-inner"
              placeholder="Enter your team name"
            />
          </div>

          <div className="group">
            <label className="block text-sm font-medium text-slate-300 mb-2 transition-colors group-focus-within:text-blue-400">
              Programming Language
            </label>
            <select
              id="language-select"
              value={formData.language}
              onChange={(e) => updateField('language', e.target.value)}
              className="w-full px-4 py-3 bg-slate-900/50 border border-slate-700/50 rounded-xl text-white transition-all duration-300 focus:outline-none focus:border-blue-500/50 focus:ring-1 focus:ring-blue-500/50 focus:bg-slate-900 shadow-inner appearance-none cursor-pointer"
            >
              <option value="cpp">C++</option>
              <option value="rust">Rust</option>
              <option value="go">Go</option>
              <option value="python">Python</option>
            </select>
          </div>

          <div className="group">
            <label className="block text-sm font-medium text-slate-300 mb-2 transition-colors group-focus-within:text-blue-400">
              Dockerfile (optional)
            </label>
            <textarea
              id="dockerfile-input"
              value={formData.dockerfile}
              onChange={(e) => updateField('dockerfile', e.target.value)}
              rows={6}
              className="w-full px-4 py-3 bg-slate-900/50 border border-slate-700/50 rounded-xl text-slate-300 placeholder-slate-600 transition-all duration-300 focus:outline-none focus:border-blue-500/50 focus:ring-1 focus:ring-blue-500/50 focus:bg-slate-900 shadow-inner font-mono text-sm resize-y"
              placeholder={'FROM ubuntu:22.04\nRUN apt-get update && apt-get install -y build-essential\nCOPY . /app\nWORKDIR /app\nRUN make\nCMD [./exchange]'}
            />
          </div>

          <div className="group">
            <label className="block text-sm font-medium text-slate-300 mb-2 transition-colors group-focus-within:text-blue-400">
              Benchmark Preset
            </label>
            <select
              id="preset-select"
              value={formData.benchmarkPreset}
              onChange={(e) => updateField('benchmarkPreset', e.target.value)}
              className="w-full px-4 py-3 bg-slate-900/50 border border-slate-700/50 rounded-xl text-white transition-all duration-300 focus:outline-none focus:border-blue-500/50 focus:ring-1 focus:ring-blue-500/50 focus:bg-slate-900 shadow-inner appearance-none cursor-pointer"
            >
              <option value="low_volatility">Low Volatility Simulation</option>
              <option value="medium_traffic">Medium Market Traffic</option>
              <option value="high_frequency_burst">High-Frequency Burst</option>
              <option value="market_open_chaos">Market Open Chaos</option>
              <option value="flash_crash">Flash Crash Simulation</option>
              <option value="stress_overload">Stress Overload Benchmark</option>
              <option value="custom">Custom Benchmark Config</option>
            </select>
          </div>

          <div className="group">
            <label className="block text-sm font-medium text-slate-300 mb-2 transition-colors group-hover:text-blue-400">Code Archive</label>
            <div
              id="file-drop-zone"
              className={`border-2 border-dashed rounded-xl p-10 text-center transition-all duration-300 cursor-pointer relative overflow-hidden ${
                isDragging
                  ? 'border-blue-500 bg-blue-500/10 shadow-[0_0_30px_rgba(59,130,246,0.15)]'
                  : 'border-slate-700/80 bg-slate-900/30 hover:border-blue-500/50 hover:bg-slate-800/50'
              }`}
              onDrop={handleDrop}
              onDragEnter={handleDragEnter}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onClick={() => document.getElementById('file-input')?.click()}
            >
              <div className={`transition-transform duration-300 ${isDragging ? 'scale-110' : 'scale-100'}`}>
                <Upload className={`h-12 w-12 mx-auto mb-4 ${isDragging ? 'text-blue-400' : 'text-slate-500 group-hover:text-blue-400/70'}`} />
              </div>
              <p className="text-slate-300 font-medium mb-2">Drag and drop your code archive here</p>
              <p className="text-slate-500 text-sm">or click to browse (ZIP, TAR.GZ)</p>
              <input
                id="file-input"
                type="file"
                className="hidden"
                accept=".zip,.tar,.tar.gz"
                onChange={handleFileInput}
              />
              {selectedFile && (
                <div className="mt-6 inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-sm text-emerald-400">
                  <div className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></div>
                  <span className="font-medium truncate max-w-[200px]">{selectedFile.name}</span>
                  <span className="opacity-75">
                    ({selectedFile.size === 0 ? '0KB' : selectedFile.size < 1024 ? '<1KB' : `${Math.round(selectedFile.size / 1024)}KB`})
                  </span>
                </div>
              )}
            </div>
          </div>

          <button
            id="submit-button"
            type="submit"
            disabled={submitting}
            className="w-full mt-4 px-6 py-4 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 disabled:from-slate-700 disabled:to-slate-800 disabled:text-slate-500 disabled:cursor-not-allowed text-white font-bold tracking-wide rounded-xl transition-all duration-300 shadow-[0_0_20px_rgba(59,130,246,0.3)] hover:shadow-[0_0_30px_rgba(59,130,246,0.5)] active:scale-[0.98] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-[#09090b] uppercase"
          >
            {submitting ? (
              <div className="flex items-center justify-center gap-3">
                <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
                <span>Submitting...</span>
              </div>
            ) : (
              'Submit for Benchmarking'
            )}
          </button>
        </form>
      </div>
    </div>
  )
}

// --- Helpers ---

function readFileAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      // Remove the data:xxx;base64, prefix.
      resolve(result.split(',')[1])
    }
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

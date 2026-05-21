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
    contestantId: '',
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
      form.append('contestant_id', formData.contestantId)
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
      localStorage.setItem('iicpc_contestant_id', formData.contestantId)
      localStorage.setItem('iicpc_team_name', formData.teamName)

      // Reset form.
      setFormData({ contestantId: '', teamName: '', language: 'cpp', dockerfile: '', benchmarkPreset: 'medium_traffic' })
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
    <div className="max-w-2xl mx-auto">
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl border border-slate-700 p-8">
        <div className="flex items-center gap-3 mb-6">
          <Upload className="h-6 w-6 text-blue-400" />
          <h2 className="text-xl font-semibold text-white">Submit Your Trading Engine</h2>
        </div>

        {/* Inline status message */}
        {message && (
          <div
            className={`mb-6 px-4 py-3 rounded-lg text-sm font-medium ${
              message.type === 'success'
                ? 'bg-green-600/20 text-green-400 border border-green-500/30'
                : 'bg-red-600/20 text-red-400 border border-red-500/30'
            }`}
          >
            {message.text}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Contestant ID</label>
            <input
              id="contestant-id"
              type="text"
              required
              value={formData.contestantId}
              onChange={(e) => updateField('contestantId', e.target.value)}
              className="w-full px-4 py-2 bg-slate-900/50 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Enter your contestant ID"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Team Name</label>
            <input
              id="team-name"
              type="text"
              required
              value={formData.teamName}
              onChange={(e) => updateField('teamName', e.target.value)}
              className="w-full px-4 py-2 bg-slate-900/50 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Enter your team name"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Programming Language
            </label>
            <select
              id="language-select"
              value={formData.language}
              onChange={(e) => updateField('language', e.target.value)}
              className="w-full px-4 py-2 bg-slate-900/50 border border-slate-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="cpp">C++</option>
              <option value="rust">Rust</option>
              <option value="go">Go</option>
              <option value="python">Python</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Dockerfile (optional)
            </label>
            <textarea
              id="dockerfile-input"
              value={formData.dockerfile}
              onChange={(e) => updateField('dockerfile', e.target.value)}
              rows={6}
              className="w-full px-4 py-2 bg-slate-900/50 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
              placeholder={'FROM ubuntu:22.04\nRUN apt-get update && apt-get install -y build-essential\nCOPY . /app\nWORKDIR /app\nRUN make\nCMD [./exchange]'}
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Benchmark Preset
            </label>
            <select
              id="preset-select"
              value={formData.benchmarkPreset}
              onChange={(e) => updateField('benchmarkPreset', e.target.value)}
              className="w-full px-4 py-2 bg-slate-900/50 border border-slate-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
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

          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Code Archive</label>
            <div
              id="file-drop-zone"
              className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors cursor-pointer ${
                isDragging
                  ? 'border-blue-500 bg-blue-500/10'
                  : 'border-slate-600 hover:border-blue-500'
              }`}
              onDrop={handleDrop}
              onDragEnter={handleDragEnter}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onClick={() => document.getElementById('file-input')?.click()}
            >
              <Upload className="h-12 w-12 text-slate-500 mx-auto mb-4" />
              <p className="text-slate-400 mb-2">Drag and drop your code archive here</p>
              <p className="text-slate-500 text-sm">or click to browse (ZIP, TAR.GZ)</p>
              <input
                id="file-input"
                type="file"
                className="hidden"
                accept=".zip,.tar,.tar.gz"
                onChange={handleFileInput}
              />
              {selectedFile && (
                <div className="mt-4 text-sm text-green-400">
                  Selected: {selectedFile.name} ({(selectedFile.size / 1024 / 1024).toFixed(2)} MB)
                </div>
              )}
            </div>
          </div>

          <button
            id="submit-button"
            type="submit"
            disabled={submitting}
            className="w-full px-6 py-3 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-slate-900"
          >
            {submitting ? 'Submitting...' : 'Submit for Benchmarking'}
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

import { NextResponse } from 'next/server'

export async function GET(
  request: Request,
  { params }: { params: { id: string } }
) {
  // Simulate latency
  await new Promise((resolve) => setTimeout(resolve, 500))

  return NextResponse.json({
    id: params.id,
    logs: [
      { timestamp: new Date(Date.now() - 10000).toISOString(), level: 'info', message: 'Starting build process...' },
      { timestamp: new Date(Date.now() - 8000).toISOString(), level: 'info', message: 'Fetching base image ubuntu:22.04' },
      { timestamp: new Date(Date.now() - 5000).toISOString(), level: 'info', message: 'Running make...' },
      { timestamp: new Date(Date.now() - 2000).toISOString(), level: 'info', message: 'Build successful. Binary created at /app/exchange' },
      { timestamp: new Date().toISOString(), level: 'info', message: 'Starting container sandbox...' },
    ]
  })
}

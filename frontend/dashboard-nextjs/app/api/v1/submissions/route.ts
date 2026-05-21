import { NextResponse } from 'next/server'

// Mock Data
const MOCK_SUBMISSIONS = [
  {
    id: 'sub_1234567890',
    team_name: 'Quant Ninjas',
    language: 'cpp',
    status: 'completed',
    created_at: new Date(Date.now() - 3600000).toISOString(),
    benchmark_results: [
      {
        metrics: {
          total_orders_sent: 120000,
          avg_latency_ms: 1.2,
          correctness: 100,
        },
        score: 95.5,
      },
    ],
  },
  {
    id: 'sub_0987654321',
    team_name: 'Rustaceans',
    language: 'rust',
    status: 'benchmarking',
    created_at: new Date().toISOString(),
    benchmark_results: [],
  },
]

export async function GET() {
  return NextResponse.json({ submissions: MOCK_SUBMISSIONS })
}

export async function POST(request: Request) {
  try {
    const formData = await request.formData()
    const teamName = formData.get('team_name')
    
    // Simulate latency
    await new Promise((resolve) => setTimeout(resolve, 1000))
    
    return NextResponse.json({
      id: `sub_${Math.random().toString(36).substr(2, 9)}`,
      status: 'pending',
      team_name: teamName || 'Unknown Team',
    })
  } catch (error) {
    return NextResponse.json(
      { error: 'Failed to process submission' },
      { status: 400 }
    )
  }
}

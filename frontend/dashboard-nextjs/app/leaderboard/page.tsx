import { Leaderboard } from '../components/Leaderboard'

export const metadata = {
  title: 'Leaderboard | IICPC',
}

export default function LeaderboardPage() {
  return (
    <div className="py-8">
      <Leaderboard />
    </div>
  )
}

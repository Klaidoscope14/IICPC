import { SubmissionsList } from '../components/SubmissionsList'

export const metadata = {
  title: 'My Submissions | IICPC',
}

export default function BenchmarksPage() {
  return (
    <div className="py-8">
      <SubmissionsList />
    </div>
  )
}

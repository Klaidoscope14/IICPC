import { SubmissionForm } from '../../components/SubmissionForm'

export const metadata = {
  title: 'Submit Code | IICPC',
}

export default function UploadPage() {
  return (
    <div className="py-8">
      <SubmissionForm />
    </div>
  )
}

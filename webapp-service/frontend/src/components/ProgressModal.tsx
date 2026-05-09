import Modal from './Modal'
import Dialog from './Dialog'

interface Props {
  title: string
  message?: string
}

export default function ProgressModal({ title, message }: Props) {
    console.log("PROGRESS MODAL")
  return (
    <Modal onClose={() => {}} canClose={false}>
      <Dialog title={title} onClose={() => {}} canClose={false}>
        <div className="flex items-center gap-3 px-5 py-6">
          <div className="w-5 h-5 rounded-full border-2 border-purple-400 border-t-transparent animate-spin shrink-0" />
          {message && <p className="text-sm text-gray-300">{message}</p>}
        </div>
      </Dialog>
    </Modal>
  )
}

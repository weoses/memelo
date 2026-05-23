import Modal from './Modal'
import Dialog from './Dialog'

interface Props {
  onClose: () => void
  onConfirm: () => void
}

export default function DeleteConfirmDialog({ onClose, onConfirm }: Props) {
  return (
    <Modal onClose={onClose}>
      <Dialog title="Delete meme?" onClose={onClose} className="w-full max-w-sm">
        <div className="p-5 space-y-4">
          <p className="text-sm text-gray-400">This action cannot be undone.</p>
          <div className="flex gap-3">
            <button
              onClick={onClose}
              className="flex-1 py-2.5 rounded-lg border border-white/15 text-sm text-gray-300
                         hover:border-white/30 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={onConfirm}
              className="flex-1 py-2.5 rounded-lg bg-red-600 hover:bg-red-700 text-sm text-white font-medium transition-colors"
            >
              Delete
            </button>
          </div>
        </div>
      </Dialog>
    </Modal>
  )
}

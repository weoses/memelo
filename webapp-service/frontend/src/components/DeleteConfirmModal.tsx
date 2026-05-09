import Modal from './Modal'
import Dialog from './Dialog'

interface Props {
  onClose: () => void
  onConfirm: () => void
}

export default function DeleteConfirmModal({ onClose, onConfirm }: Props) {
  return (
    <Modal onClose={onClose}>
      <Dialog title="Delete media?" onClose={onClose}>
        <div className="px-5 py-4">
        <p className="text-gray-300 text-sm">This cannot be undone.</p>
        <div className="flex gap-2 mt-5 justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm rounded-lg bg-gray-700 hover:bg-gray-600 text-white"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className="px-4 py-2 text-sm rounded-lg bg-red-600 hover:bg-red-500 text-white"
          >
            Delete
          </button>
        </div>
        </div>
      </Dialog>
    </Modal>
  )
}

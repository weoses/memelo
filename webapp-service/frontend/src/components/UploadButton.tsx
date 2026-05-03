interface Props {
  onOpen: () => void
}

export default function UploadButton({ onOpen }: Props) {
  return (
    <button
      onClick={onOpen}
      className="shrink-0 rounded-lg bg-purple-600 hover:bg-purple-700
                 px-3 py-2 text-sm font-medium text-white transition-colors"
    >
      + Upload
    </button>
  )
}
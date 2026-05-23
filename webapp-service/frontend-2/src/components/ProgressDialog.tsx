interface Props {
  title: string
  message?: string
}

export default function ProgressDialog({ title, message }: Props) {
  return (
    <div className="fixed inset-0 z-[70] bg-black/70 flex items-center justify-center p-4">
      <div className="bg-[#1a1a1a] border border-white/[0.08] rounded-xl px-8 py-7 flex flex-col items-center gap-4 max-w-xs w-full">
        <div className="w-8 h-8 rounded-full border-2 border-white/20 border-t-white animate-spin" />
        <div className="text-center">
          <p className="text-sm font-semibold text-white">{title}</p>
          {message && <p className="text-xs text-gray-500 mt-1">{message}</p>}
        </div>
      </div>
    </div>
  )
}

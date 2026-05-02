import { useRef, useState, DragEvent } from 'react'
import { getUploadUrl, uploadToS3Post, parseByToken, Meme } from '../api/client'
import Modal from './Modal'
import Dialog from './Dialog'

type Phase = 'idle' | 'uploading' | 'success' | 'duplicate' | 'error'

interface Props {
  onClose: () => void
  onUploaded: (meme: Meme) => void
}

export default function UploadModal({ onClose, onUploaded }: Props) {
  const [phase, setPhase] = useState<Phase>('idle')
  const [progress, setProgress] = useState(0)
  const [errorMsg, setErrorMsg] = useState('')
  const [dragging, setDragging] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const canClose = phase !== 'uploading'

  const startUpload = async (file: File) => {
    setPhase('uploading')
    setProgress(0)
    try {
      const { upload_url, form_fields, token } = await getUploadUrl(file.name, file.type, file.size)
      await uploadToS3Post(upload_url, form_fields, file, setProgress)
      const meme = await parseByToken(token)
      if (meme.status === 'duplicate') {
        setPhase('duplicate')
      } else {
        setPhase('success')
        onUploaded(meme)
        setTimeout(onClose, 1500)
      }
    } catch (e) {
      setErrorMsg(e instanceof Error ? e.message : String(e))
      setPhase('error')
    }
  }

  const onDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setDragging(false)
    const file = e.dataTransfer.files[0]
    if (file) startUpload(file)
  }

  return (
    <Modal onClose={onClose} canClose={canClose}>
      <Dialog title="Upload meme" onClose={onClose} canClose={canClose} className="w-full max-w-md overflow-hidden">
        <div className="p-5">
          {/* Idle — drop zone */}
          {phase === 'idle' && (
            <div
              onDragOver={e => { e.preventDefault(); setDragging(true) }}
              onDragLeave={e => { if (!e.currentTarget.contains(e.relatedTarget as Node)) setDragging(false) }}
              onDrop={onDrop}
              onClick={() => inputRef.current?.click()}
              className={`flex flex-col items-center justify-center gap-3 rounded-xl border-2 border-dashed
                          cursor-pointer py-12 transition-colors
                          ${dragging
                            ? 'border-purple-400 bg-purple-950/40'
                            : 'border-gray-700 hover:border-purple-500 hover:bg-gray-800/50'}`}
            >
              <span className="text-4xl select-none">🖼️</span>
              <p className="text-sm text-gray-300 font-medium">
                Drop a file here or <span className="text-purple-400 underline">browse</span>
              </p>
              <p className="text-xs text-gray-500">Images and videos up to 50 MB</p>
              <input
                ref={inputRef}
                type="file"
                accept="image/*,video/*"
                className="hidden"
                onChange={e => {
                  const f = e.target.files?.[0]
                  if (f) startUpload(f)
                  e.target.value = ''
                }}
              />
            </div>
          )}

          {/* Uploading — progress bar */}
          {phase === 'uploading' && (
            <div className="flex flex-col items-center gap-4 py-8">
              <p className="text-sm text-gray-300">Uploading… {progress}%</p>
              <div className="w-full bg-gray-700 rounded-full h-3 overflow-hidden">
                <div
                  className="h-full bg-purple-500 rounded-full transition-all duration-200"
                  style={{ width: `${progress}%` }}
                />
              </div>
              <p className="text-xs text-gray-500">Processing may take a moment after upload</p>
            </div>
          )}

          {/* Success */}
          {phase === 'success' && (
            <div className="flex flex-col items-center gap-3 py-8">
              <span className="text-5xl">✓</span>
              <p className="text-sm text-green-400 font-medium">Upload complete!</p>
            </div>
          )}

          {/* Duplicate */}
          {phase === 'duplicate' && (
            <div className="flex flex-col items-center gap-3 py-8">
              <span className="text-5xl">⊘</span>
              <p className="text-sm text-yellow-400 font-medium">Already in your library</p>
              <p className="text-xs text-gray-500">This meme is a duplicate of one you already have.</p>
            </div>
          )}

          {/* Error */}
          {phase === 'error' && (
            <div className="flex flex-col items-center gap-4 py-8">
              <span className="text-5xl">✗</span>
              <p className="text-sm text-red-400 font-medium">Upload failed</p>
              <p className="text-xs text-gray-500 text-center">{errorMsg}</p>
              <button
                onClick={() => setPhase('idle')}
                className="mt-1 rounded-lg bg-purple-600 hover:bg-purple-700 px-4 py-2 text-sm font-medium text-white"
              >
                Try again
              </button>
            </div>
          )}
        </div>
      </Dialog>
    </Modal>
  )
}
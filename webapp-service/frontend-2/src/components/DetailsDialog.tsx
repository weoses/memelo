import {useEffect, useState, useMemo, useCallback} from 'react'
import {Meme} from '../api/client'
import {useMediaStore} from '../store/mediaStore'
import Modal from './Modal'
import Dialog from './Dialog'
import EditMetadataDialog from './EditMetadataDialog'
import DeleteConfirmDialog from './DeleteConfirmDialog'

interface Props {
    index: number
    onClose: () => void
    onPrev?: () => void
    onNext?: () => void
    onRecompute?: () => void
    onDownload?: () => void
    onDelete?: () => void
}

function formatDate(sortingId: number): string {
    if (!sortingId) return '—'
    const ms = sortingId > 1e12 ? sortingId : sortingId * 1000
    return new Date(ms).toLocaleDateString('en-US', {month: 'short', day: 'numeric', year: 'numeric'})
}

function InfoGrid({meme}: { meme: Meme }) {
    const date = useMemo(() => formatDate(meme.sorting_id), [meme.sorting_id])
    const format = meme.type ? meme.type.toUpperCase() : '—'
    return (
        <div className="space-y-3">
            <div>
                <p className="text-xs text-gray-600 mb-1">ID</p>
                <IdRow id={meme.id}/>
            </div>
            <div className="grid grid-cols-3 gap-3">
                {[
                    {label: 'Uploaded', value: date},
                    {label: 'Size', value: '—'},
                    {label: 'Format', value: format},
                ].map(({label, value}) => (
                    <div key={label}>
                        <p className="text-xs text-gray-600 mb-0.5">{label}</p>
                        <p className="text-sm font-semibold text-white">{value}</p>
                    </div>
                ))}
            </div>
        </div>
    )
}

function Section({label, children}: { label: string; children: React.ReactNode }) {
    return (
        <div className="bg-[#1c1c1c] rounded-xl p-4 space-y-2">
            <p className="text-[10px] font-semibold text-gray-500 uppercase tracking-widest flex items-center gap-1.5">
                {label}
            </p>
            {children}
        </div>
    )
}

function IdRow({id}: { id: string }) {
    const [copied, setCopied] = useState(false)
    const copy = useCallback(() => {
        navigator.clipboard.writeText(id).then(() => {
            setCopied(true)
            setTimeout(() => setCopied(false), 1500)
        })
    }, [id])
    return (
        <div className="flex items-center justify-between gap-2 min-w-0">
            <p className="text-xs font-mono text-gray-500 truncate">{id}</p>
            <button
                onClick={copy}
                className="shrink-0 text-gray-600 hover:text-gray-300 transition-colors"
                aria-label="Copy ID"
            >
                {copied ? (
                    <svg width="13" height="13" viewBox="0 0 20 20" fill="none">
                        <path d="M4 10l5 5 7-7" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"
                              strokeLinejoin="round"/>
                    </svg>
                ) : (
                    <svg width="13" height="13" viewBox="0 0 20 20" fill="none">
                        <rect x="7" y="7" width="10" height="10" rx="1.5" stroke="currentColor" strokeWidth="1.7"/>
                        <path d="M13 7V5a1.5 1.5 0 0 0-1.5-1.5h-7A1.5 1.5 0 0 0 3 5v7A1.5 1.5 0 0 0 4.5 13.5H7"
                              stroke="currentColor" strokeWidth="1.7" strokeLinecap="round"/>
                    </svg>
                )}
            </button>
        </div>
    )
}

function TagList({tags}: { tags: string[] }) {
    if (!tags || tags.length === 0) return <p className="text-sm text-gray-600 italic">no tags</p>
    return (
        <div className="flex flex-wrap gap-1.5">
            {tags.map(t => (
                <span key={t} className="border border-white/15 rounded-full px-2.5 py-0.5 text-xs text-gray-400">
          #{t}
        </span>
            ))}
        </div>
    )
}

function MediaArea({meme, onPrev, onNext}: { meme: Meme; onPrev?: () => void; onNext?: () => void }) {
    const isVideo = meme.type === 'video'
    const [loaded, setLoaded] = useState(false)

    useEffect(() => {
        if (!isVideo) setLoaded(false)
    }, [meme.id, isVideo])

    return (
        <div className="relative w-full h-full bg-black flex items-center justify-center overflow-hidden">
            {!loaded && !isVideo && (
                <div className="absolute inset-0 flex items-center justify-center">
                    <div className="w-7 h-7 border-2 border-white/20 border-t-white/60 rounded-full animate-spin"/>
                </div>
            )}
            {isVideo ? (
                <video
                    key={meme.id}
                    src={meme.original_url}
                    controls
                    autoPlay
                    className="w-full h-full object-contain"
                />
            ) : (
                <img
                    key={meme.id}
                    src={meme.original_url || meme.thumbnail_url}
                    alt={meme.caption || 'meme'}
                    className="w-full h-full object-contain"
                    style={{opacity: loaded ? 1 : 0, transition: 'opacity 0.2s'}}
                    onLoad={() => setLoaded(true)}
                />
            )}

            {onPrev && (
                <button
                    onClick={e => {
                        e.stopPropagation();
                        onPrev()
                    }}
                    className="absolute left-3 top-1/2 -translate-y-1/2 w-9 h-9 rounded-full bg-black/60
                     hover:bg-black/80 text-white flex items-center justify-center transition-colors"
                    aria-label="Previous"
                >
                    <svg width="18" height="18" viewBox="0 0 20 20" fill="none">
                        <path d="M13 4l-6 6 6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round"
                              strokeLinejoin="round"/>
                    </svg>
                </button>
            )}
            {onNext && (
                <button
                    onClick={e => {
                        e.stopPropagation();
                        onNext()
                    }}
                    className="absolute right-3 top-1/2 -translate-y-1/2 w-9 h-9 rounded-full bg-black/60
                     hover:bg-black/80 text-white flex items-center justify-center transition-colors"
                    aria-label="Next"
                >
                    <svg width="18" height="18" viewBox="0 0 20 20" fill="none">
                        <path d="M7 4l6 6-6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round"
                              strokeLinejoin="round"/>
                    </svg>
                </button>
            )}
        </div>
    )
}

function MetadataPanel({
                           meme,
                           onClose,
                           onEditMetadata,
                           onDelete,
                           onRecompute,
                       }: {
    meme: Meme
    onClose: () => void
    onEditMetadata: () => void
    onDelete: () => void
    onRecompute?: () => void
}) {
    const [moreOpen, setMoreOpen] = useState(false)
    const ocrText = meme.on_screen_text || meme.ocr_result

    return (
        <div className="flex flex-col h-full bg-[#141414]">
            {/* Header */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-white/[0.08] shrink-0">
                <div className="flex items-center gap-2 min-w-0">
                    <h2 className="text-sm font-semibold text-white truncate">Meme Details</h2>
                    {meme.edited && (
                        <span className="shrink-0 text-xs border border-white/20 rounded px-1.5 py-0.5 text-gray-400">Edited</span>
                    )}
                </div>
                <div className="flex items-center gap-1 shrink-0">
                    <button
                        onClick={() => window.open(meme.original_url, '_blank')}
                        className="text-gray-400 hover:text-white transition-colors p-1.5"
                        aria-label="Share"
                    >
                        <svg width="17" height="17" viewBox="0 0 20 20" fill="none">
                            <path d="M10 3v10M6 7l4-4 4 4" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round"/>
                            <path d="M4 13v3a1 1 0 001 1h10a1 1 0 001-1v-3" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round"/>
                        </svg>
                    </button>
                    <button
                        onClick={onClose}
                        className="text-gray-500 hover:text-white transition-colors text-xl leading-none p-1"
                        aria-label="Close"
                    >
                        ×
                    </button>
                </div>
            </div>

            {/* Scrollable content */}
            <div className="flex-1 overflow-y-auto p-4 space-y-3 min-h-0">
                <Section label="Caption">
                    <p className={`text-sm leading-relaxed ${meme.caption ? 'text-white' : 'text-gray-600 italic'}`}>
                        {meme.caption || 'No caption'}
                    </p>
                </Section>
                <Section label="Information">
                    <InfoGrid meme={meme}/>
                </Section>

                <Section label="Tags">
                    <TagList tags={meme.tags}/>
                </Section>


                {ocrText && (
                    <Section label="OCR Text">
                        <div className="bg-black rounded-lg p-3 overflow-x-auto">
                            <p className="text-sm font-mono text-green-400 whitespace-pre-wrap break-words leading-relaxed">
                                {ocrText}
                            </p>
                        </div>
                    </Section>
                )}

                {meme.audio_transcript && (
                    <Section label="Audio transcript">
                        <div className="bg-black rounded-lg p-3 overflow-x-auto">
                            <p className="text-sm font-mono text-green-400 whitespace-pre-wrap break-words leading-relaxed">
                                {meme.audio_transcript}
                            </p>
                        </div>
                    </Section>
                )}

                {meme.audio_track && (
                    <Section label="Audio track">
                        <p className="text-sm text-gray-300">{meme.audio_track}</p>
                    </Section>
                )}


            </div>

            {/* Sticky bottom bar */}
            <div className="shrink-0 px-4 py-4 border-t border-white/[0.08] flex gap-2">
                <button
                    onClick={onEditMetadata}
                    className="flex-1 flex items-center justify-center gap-2 py-2.5 rounded-lg bg-white text-black
                     text-sm font-semibold hover:bg-gray-100 transition-colors"
                >
                    <svg width="14" height="14" viewBox="0 0 20 20" fill="none">
                        <path d="M14.85 2.15a2.12 2.12 0 0 1 3 3L6.5 16.5l-4 1 1-4L14.85 2.15z"
                              stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"/>
                    </svg>
                    Edit Metadata
                </button>
                <button
                    onClick={onDelete}
                    className="flex items-center justify-center gap-1.5 px-3 py-2.5 rounded-lg border border-red-500/30
                     text-red-400 hover:bg-red-500/10 text-sm transition-colors"
                    aria-label="Delete"
                >
                    <svg width="15" height="15" viewBox="0 0 20 20" fill="none">
                        <path d="M4 6h12M8 6V4h4v2M9 10v5M11 10v5M5 6l1 11h8l1-11" stroke="currentColor"
                              strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round"/>
                    </svg>
                </button>
                <div className="relative flex">
                    <button
                        onClick={() => setMoreOpen(o => !o)}
                        className="flex items-center justify-center px-3 rounded-lg border border-white/10
                         text-gray-400 hover:border-white/25 hover:text-white text-sm transition-colors self-stretch"
                        aria-label="More options"
                    >
                        <svg width="15" height="15" viewBox="0 0 20 20" fill="none">
                            <circle cx="4" cy="10" r="1.3" fill="currentColor"/>
                            <circle cx="10" cy="10" r="1.3" fill="currentColor"/>
                            <circle cx="16" cy="10" r="1.3" fill="currentColor"/>
                        </svg>
                    </button>
                    {moreOpen && (
                        <div className="absolute right-0 bottom-full mb-2 w-44 bg-[#1e1e1e] border border-white/10
                                        rounded-xl shadow-2xl overflow-hidden z-10">
                            {onRecompute && (
                                <button
                                    onClick={() => { setMoreOpen(false); onRecompute() }}
                                    className="w-full text-left px-4 py-2.5 text-sm text-gray-300 hover:bg-white/5 transition-colors"
                                >
                                    Recompute
                                </button>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    )
}

export default function DetailsDialog({index, onClose, onPrev, onNext, onRecompute, onDelete}: Props) {
    const meme = useMediaStore(s => index >= 0 ? s.memes[index] : s.detachedMeme)
    const updateMemeInStore = useMediaStore(s => s.updateMeme)

    const [editOpen, setEditOpen] = useState(false)
    const [deleteOpen, setDeleteOpen] = useState(false)
    const [mobileMoreOpen, setMobileMoreOpen] = useState(false)
    const [isDesktop, setIsDesktop] = useState(() => window.matchMedia('(min-width: 640px)').matches)

    useEffect(() => {
        const mq = window.matchMedia('(min-width: 640px)')
        const handler = (e: MediaQueryListEvent) => setIsDesktop(e.matches)
        mq.addEventListener('change', handler)
        return () => mq.removeEventListener('change', handler)
    }, [])

    useEffect(() => {
        const handler = (e: KeyboardEvent) => {
            if (e.key === 'ArrowLeft') onPrev?.()
            if (e.key === 'ArrowRight') onNext?.()
        }
        window.addEventListener('keydown', handler)
        return () => window.removeEventListener('keydown', handler)
    }, [onPrev, onNext])

    if (!meme) return null

    const handleDeleteConfirm = () => {
        setDeleteOpen(false)
        onDelete?.()
        onClose()
    }

    // Desktop: side-by-side fullscreen overlay (not using Modal/Dialog to keep custom layout)
    const desktopView = (
        <div className="flex fixed inset-0 z-50" onClick={onClose}>
            <div
                className="flex w-full max-w-6xl mx-auto my-auto h-[90vh] rounded-2xl overflow-hidden shadow-2xl border border-white/[0.06]"
                onClick={e => e.stopPropagation()}
            >
                {/* Left: media */}
                <div className="flex-[3] min-w-0 bg-black relative">
                    {/* Desktop prev/next outside the dialog */}
                    {onPrev && (
                        <button
                            onClick={e => {
                                e.stopPropagation();
                                onPrev()
                            }}
                            className="absolute left-3 top-1/2 -translate-y-1/2 z-10 w-10 h-10 rounded-full bg-black/60
                         hover:bg-black/80 text-white flex items-center justify-center transition-colors"
                            aria-label="Previous"
                        >
                            <svg width="18" height="18" viewBox="0 0 20 20" fill="none">
                                <path d="M13 4l-6 6 6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round"
                                      strokeLinejoin="round"/>
                            </svg>
                        </button>
                    )}
                    {onNext && (
                        <button
                            onClick={e => {
                                e.stopPropagation();
                                onNext()
                            }}
                            className="absolute right-3 top-1/2 -translate-y-1/2 z-10 w-10 h-10 rounded-full bg-black/60
                         hover:bg-black/80 text-white flex items-center justify-center transition-colors"
                            aria-label="Next"
                        >
                            <svg width="18" height="18" viewBox="0 0 20 20" fill="none">
                                <path d="M7 4l6 6-6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round"
                                      strokeLinejoin="round"/>
                            </svg>
                        </button>
                    )}
                    <MediaArea meme={meme}/>
                </div>

                {/* Right: metadata panel */}
                <div className="w-[340px] xl:w-[380px] shrink-0">
                    <MetadataPanel
                        meme={meme}
                        onClose={onClose}
                        onEditMetadata={() => setEditOpen(true)}
                        onDelete={() => setDeleteOpen(true)}
                        onRecompute={onRecompute}
                    />
                </div>
            </div>

            {/* Dark backdrop outside the panel */}
            <div className="fixed inset-0 -z-10 bg-black/85"/>
        </div>
    )

    // Mobile: bottom sheet
    const mobileView = (
        <div>
            <Modal onClose={onClose}>
                <Dialog
                    onClose={onClose}
                    noHeader
                    className="w-full max-h-[92dvh] overflow-y-auto"
                    onSwipeLeft={onNext}
                    onSwipeRight={onPrev}
                >
                    {/* Mobile header */}
                    <div className="flex items-center justify-between px-4 py-3 border-b border-white/[0.08]">
                        <button onClick={onClose}
                                className="text-gray-400 hover:text-white flex items-center gap-1 text-sm transition-colors">
                            <svg width="18" height="18" viewBox="0 0 20 20" fill="none">
                                <path d="M13 4l-6 6 6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round"
                                      strokeLinejoin="round"/>
                            </svg>
                        </button>
                        <h2 className="text-sm font-semibold text-white">Meme Details</h2>
                        <button
                            onClick={() => window.open(meme.original_url, '_blank')}
                            className="text-gray-400 hover:text-white transition-colors"
                            aria-label="Share"
                        >
                            <svg width="18" height="18" viewBox="0 0 20 20" fill="none">
                                <path d="M10 3v10M6 7l4-4 4 4" stroke="currentColor" strokeWidth="1.7"
                                      strokeLinecap="round" strokeLinejoin="round"/>
                                <path d="M4 13v3a1 1 0 001 1h10a1 1 0 001-1v-3" stroke="currentColor" strokeWidth="1.7"
                                      strokeLinecap="round"/>
                            </svg>
                        </button>
                    </div>

                    {/* Image */}
                    <div className="relative bg-black" style={{height: Math.round(window.innerWidth * 0.6)}}>
                        <MediaArea meme={meme} onPrev={onPrev} onNext={onNext}/>
                    </div>

                    {/* Sections */}
                    <div className="p-4 space-y-3">
                        <Section label="Caption">
                            <p className={`text-sm leading-relaxed ${meme.caption ? 'text-white' : 'text-gray-600 italic'}`}>
                                {meme.caption || 'No caption'}
                            </p>
                        </Section>

                        {(meme.on_screen_text || meme.ocr_result) && (
                            <Section label="OCR Text">
                                <div className="bg-black rounded-lg p-3">
                                    <p className="text-sm font-mono text-green-400 whitespace-pre-wrap break-words leading-relaxed">
                                        {meme.on_screen_text || meme.ocr_result}
                                    </p>
                                </div>
                            </Section>
                        )}

                        <Section label="Tags">
                            <TagList tags={meme.tags}/>
                        </Section>

                        <Section label="Information">
                            <InfoGrid meme={meme}/>
                        </Section>
                    </div>

                    {/* Bottom action bar */}
                    <div className="px-4 pb-6 pt-2 flex gap-2 border-t border-white/[0.08]">
                        <button
                            onClick={() => setEditOpen(true)}
                            className="flex-1 flex items-center justify-center gap-2 py-3 rounded-xl bg-white text-black
                         text-sm font-semibold hover:bg-gray-100 transition-colors"
                        >
                            <svg width="14" height="14" viewBox="0 0 20 20" fill="none">
                                <path d="M14.85 2.15a2.12 2.12 0 0 1 3 3L6.5 16.5l-4 1 1-4L14.85 2.15z"
                                      stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"
                                      strokeLinejoin="round"/>
                            </svg>
                            Edit
                        </button>
                        <button
                            onClick={() => setDeleteOpen(true)}
                            className="flex items-center justify-center px-4 py-3 rounded-xl border border-red-500/30
                         text-red-400 hover:bg-red-500/10 text-sm transition-colors"
                            aria-label="Delete"
                        >
                            <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
                                <path d="M4 6h12M8 6V4h4v2M9 10v5M11 10v5M5 6l1 11h8l1-11" stroke="currentColor"
                                      strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round"/>
                            </svg>
                        </button>
                        <div className="relative">
                            <button
                                onClick={() => setMobileMoreOpen(o => !o)}
                                className="flex items-center justify-center px-4 py-3 rounded-xl border border-white/10
                             text-gray-400 hover:border-white/25 hover:text-white transition-colors"
                                aria-label="More options"
                            >
                                <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
                                    <circle cx="4" cy="10" r="1.3" fill="currentColor"/>
                                    <circle cx="10" cy="10" r="1.3" fill="currentColor"/>
                                    <circle cx="16" cy="10" r="1.3" fill="currentColor"/>
                                </svg>
                            </button>
                            {mobileMoreOpen && (
                                <div className="absolute right-0 bottom-full mb-2 w-44 bg-[#1e1e1e] border border-white/10
                                                rounded-xl shadow-2xl overflow-hidden z-10">
                                    {onRecompute && (
                                        <button
                                            onClick={() => { setMobileMoreOpen(false); onRecompute() }}
                                            className="w-full text-left px-4 py-3 text-sm text-gray-300 hover:bg-white/5 transition-colors"
                                        >
                                            Recompute
                                        </button>
                                    )}
                                </div>
                            )}
                        </div>
                    </div>
                </Dialog>
            </Modal>
        </div>
    )

    return (
        <>
            {isDesktop ? desktopView : mobileView}

            {editOpen && (
                <EditMetadataDialog
                    meme={meme}
                    onClose={() => setEditOpen(false)}
                    onSaved={updated => updateMemeInStore(updated)}
                />
            )}

            {deleteOpen && (
                <DeleteConfirmDialog
                    onClose={() => setDeleteOpen(false)}
                    onConfirm={handleDeleteConfirm}
                />
            )}
        </>
    )
}

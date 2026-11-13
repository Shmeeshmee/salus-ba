import { useEffect, useRef, type KeyboardEvent, type ReactNode } from 'react'

// Dialog is a modal over the page, named by its label: it takes the focus, keeps Tab inside, closes on Escape and
// gives the focus back to what had it.
export function Dialog({ label, className, onClose, children }: { label: string; className?: string; onClose: () => void; children: ReactNode }) {
  const dialog = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const before = document.activeElement
    dialog.current?.querySelector<HTMLElement>('select, input, textarea, button, a[href]')?.focus()
    return () => {
      if (before instanceof HTMLElement && before.isConnected) before.focus()
    }
  }, [])

  function onKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault()
      onClose()
      return
    }
    if (e.key !== 'Tab') return
    const focusable = Array.from(dialog.current?.querySelectorAll<HTMLElement>('select, input, textarea, button, a[href]') ?? [])
    const first = focusable[0]
    const last = focusable[focusable.length - 1]
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault()
      last.focus()
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault()
      first.focus()
    }
  }

  return (
    <div className="viewer-backdrop">
      <div ref={dialog} className={className ?? 'chat-dialog'} role="dialog" aria-modal="true" aria-label={label} onKeyDown={onKeyDown}>
        {children}
      </div>
    </div>
  )
}

import { useEffect, useRef, type KeyboardEvent, type ReactNode } from 'react'

// focusable are the controls Tab reaches: a disabled one is skipped, so the last enabled control wraps to the first.
const focusable = 'select:not(:disabled), input:not(:disabled), textarea:not(:disabled), button:not(:disabled), a[href]'

// Dialog is a modal over the page, named by its label: it takes the focus, keeps Tab inside, closes on Escape and
// gives the focus back to what had it. Escape closes only the innermost dialog: a dialog opened from another (the patient
// card from the desk's patient search) stops the key there.
export function Dialog({ label, className, onClose, children }: { label: string; className?: string; onClose: () => void; children: ReactNode }) {
  const dialog = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const before = document.activeElement
    const own = dialog.current
    ;(own?.querySelector<HTMLElement>(focusable) ?? own)?.focus() // a dialog of text alone takes the focus itself
    return () => {
      if (before instanceof HTMLElement && before.isConnected) before.focus()
    }
  }, [])

  function onKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      onClose()
      return
    }
    if (e.key !== 'Tab') return
    const stops = Array.from(dialog.current?.querySelectorAll<HTMLElement>(focusable) ?? [])
    const first = stops[0]
    const last = stops[stops.length - 1]
    if (!first) e.preventDefault() // nothing to reach: the focus stays on the dialog
    else if (e.shiftKey && document.activeElement === first) {
      e.preventDefault()
      last.focus()
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault()
      first.focus()
    }
  }

  return (
    <div className="viewer-backdrop">
      <div ref={dialog} className={className ?? 'chat-dialog'} role="dialog" aria-modal="true" aria-label={label} tabIndex={-1} onKeyDown={onKeyDown}>
        {children}
      </div>
    </div>
  )
}

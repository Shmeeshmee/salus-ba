import { useId, useRef, useState, type KeyboardEvent, type MouseEvent, type ReactNode } from 'react'
import { useT } from './i18n/useT'
import { CrossIcon, MenuIcon } from './icons'
import { usePath } from './route'

// HeaderMenu folds the header's links and tools behind a button on a phone (readiness, after RD3): a disclosure, not a
// modal. The button says whether the panel is open and names the panel it controls; Escape closes it and gives the
// focus back to the button; choosing a link or any change of page closes it. On a wider screen app.css shows the
// panel's contents in the header as they were and hides the button, so the two widths have one set of links.
export function HeaderMenu({ children }: { children: ReactNode }) {
  const t = useT()
  const id = useId()
  const path = usePath()
  // The panel is open on the page it was opened on: a change of page closes it without an effect.
  const [openOn, setOpenOn] = useState<string>()
  const open = openOn === path
  const button = useRef<HTMLButtonElement>(null)

  function onKeyDown(e: KeyboardEvent) {
    // A dialog opened from the panel (the medical terms) handles its own Escape first.
    if (e.key !== 'Escape' || !open || e.defaultPrevented) return
    e.preventDefault()
    setOpenOn(undefined)
    button.current?.focus()
  }

  function onClick(e: MouseEvent) {
    if (e.target instanceof Element && e.target.closest('a')) setOpenOn(undefined)
  }

  return (
    <>
      <button ref={button} type="button" className="quiet menu-button" aria-expanded={open} aria-controls={id} onClick={() => setOpenOn(open ? undefined : path)} onKeyDown={onKeyDown}>
        {open ? <CrossIcon /> : <MenuIcon />}
        {t('shell.menu')}
      </button>
      <div id={id} className={open ? 'bar-menu open' : 'bar-menu'} onKeyDown={onKeyDown} onClick={onClick}>
        {children}
      </div>
    </>
  )
}

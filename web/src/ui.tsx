import { useEffect, useId, useRef, type InputHTMLAttributes, type MouseEvent, type ReactNode } from 'react'
import { Logo } from './icons'
import { DeviceLanguagePicker } from './LanguagePicker'
import { navigate } from './route'

// AuthCard frames the screens before sign-in, with the language picker kept on the device below them.
export function AuthCard({ title, children }: { title: string; children: ReactNode }) {
  return (
    <main className="auth">
      <div className="auth-card">
        <Logo />
        <h1>{title}</h1>
        {children}
        <DeviceLanguagePicker />
      </div>
    </main>
  )
}

type FieldProps = InputHTMLAttributes<HTMLInputElement> & { label: string; hint?: string }

export function Field({ label, hint, id: callerId, ...input }: FieldProps) {
  const generatedId = useId()
  const id = callerId ?? generatedId
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <input {...input} id={id} aria-describedby={hint ? `${id}-hint` : undefined} />
      {hint && (
        <p className="hint" id={`${id}-hint`}>
          {hint}
        </p>
      )}
    </div>
  )
}

export function Alert({ message }: { message?: string }) {
  if (!message) return null
  return (
    <p className="alert" role="alert">
      {message}
    </p>
  )
}

// TipTag is a tag with a tooltip, shown on hover and on focus.
export function TipTag({ className, label, tip }: { className: string; label: string; tip: string }) {
  const id = useId()
  return (
    <span className="tip">
      <span className={`tag ${className}`} tabIndex={0} aria-describedby={id}>
        {label}
      </span>
      <span role="tooltip" id={id}>
        {tip}
      </span>
    </span>
  )
}

export function Notice({ message }: { message?: string }) {
  if (!message) return null
  return (
    <p className="notice" role="status">
      {message}
    </p>
  )
}

export function Submit({ busy, label, busyLabel }: { busy: boolean; label: string; busyLabel: string }) {
  return (
    <button type="submit" className="primary" disabled={busy}>
      {busy ? busyLabel : label}
    </button>
  )
}

export function Link({ to, className, children }: { to: string; className?: string; children: ReactNode }) {
  function onClick(event: MouseEvent<HTMLAnchorElement>) {
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.button !== 0) return
    event.preventDefault()
    navigate(to)
  }
  return (
    <a href={to} className={className} onClick={onClick} aria-current={location.pathname === to ? 'page' : undefined}>
      {children}
    </a>
  )
}

type ConfirmProps = {
  question: string
  lines: string[]
  confirm: string
  cancel: string
  danger?: boolean
  busy?: boolean
  onConfirm: () => void
  onCancel: () => void
}

// Confirm asks before an action, in place on the page: an alert dialog named by its question and described by its
// lines. It takes the focus on Cancel, the button that changes nothing, and gives it back to what had it before
// when it closes, unless the focus has gone elsewhere or that element is gone.
export function Confirm({ question, lines, confirm, cancel, danger, busy, onConfirm, onCancel }: ConfirmProps) {
  const id = useId()
  const dialog = useRef<HTMLDivElement>(null)
  const cancelButton = useRef<HTMLButtonElement>(null)
  useEffect(() => {
    const before = document.activeElement
    const own = dialog.current
    cancelButton.current?.focus()
    return () => {
      const now = document.activeElement
      const lost = !now || now === document.body || own?.contains(now)
      if (lost && before instanceof HTMLElement && before.isConnected) before.focus()
    }
  }, [])
  return (
    <div ref={dialog} className="confirm" role="alertdialog" aria-labelledby={`${id}-q`} aria-describedby={lines.length > 0 ? `${id}-d` : undefined}>
      <p className="strong" id={`${id}-q`}>
        {question}
      </p>
      <div id={`${id}-d`}>
        {lines.map((l) => (
          <p key={l}>{l}</p>
        ))}
      </div>
      <div className="actions">
        <button type="button" className={danger ? 'danger' : 'primary'} disabled={busy} onClick={onConfirm}>
          {confirm}
        </button>
        <button type="button" ref={cancelButton} disabled={busy} onClick={onCancel}>
          {cancel}
        </button>
      </div>
    </div>
  )
}

import { useId, type InputHTMLAttributes, type MouseEvent, type ReactNode } from 'react'
import { Logo } from './icons'
import { navigate } from './route'

export function AuthCard({ title, children }: { title: string; children: ReactNode }) {
  return (
    <main className="auth">
      <div className="auth-card">
        <Logo />
        <h1>{title}</h1>
        {children}
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

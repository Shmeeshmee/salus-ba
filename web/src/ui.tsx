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

export function Field({ label, hint, ...input }: FieldProps) {
  const id = useId()
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <input id={id} aria-describedby={hint ? `${id}-hint` : undefined} {...input} />
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

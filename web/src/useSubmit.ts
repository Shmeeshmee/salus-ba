import { useState, type FormEvent } from 'react'

// useSubmit runs a form's action once at a time. The action returns the message to show, or nothing.
export function useSubmit(action: () => Promise<string | undefined>) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    if (busy) return
    setBusy(true)
    setError(undefined)
    const message = await action()
    setError(message)
    setBusy(false)
  }

  return { busy, error, onSubmit }
}

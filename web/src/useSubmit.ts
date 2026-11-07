import { useState, type FormEvent } from 'react'
import { useT } from './i18n/useT'

// useSubmit runs a form's action once at a time. The action returns the message to show, or nothing. An
// action that rejects (a network failure) shows the fixed message and never leaves the form disabled.
export function useSubmit(action: () => Promise<string | undefined>) {
  const t = useT()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    if (busy) return
    setBusy(true)
    setError(undefined)
    try {
      setError(await action())
    } catch {
      setError(t('common.somethingWrong'))
    } finally {
      setBusy(false)
    }
  }

  return { busy, error, onSubmit }
}

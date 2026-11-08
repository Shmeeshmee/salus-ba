import { cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderInLanguage } from './i18n/renderInLanguage'
import { useSubmit } from './useSubmit'

afterEach(cleanup)

function Form({ action }: { action: () => Promise<string | undefined> }) {
  const { busy, error, onSubmit } = useSubmit(action)
  return (
    <form onSubmit={onSubmit}>
      {error && <p role="alert">{error}</p>}
      <button type="submit" disabled={busy}>
        Save
      </button>
    </form>
  )
}

describe('useSubmit', () => {
  it('shows a fixed message and enables the form again when the action rejects', async () => {
    const action = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'))
    renderInLanguage(<Form action={action} />, 'en')

    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect((await screen.findByRole('alert')).textContent).toBe('Something went wrong. Try again.')
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Save' }).disabled).toBe(false)
  })

  it('shows the fixed message in the UI language', async () => {
    renderInLanguage(<Form action={vi.fn().mockRejectedValue(new TypeError('Failed to fetch'))} />, 'bs')

    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect((await screen.findByRole('alert')).textContent).toBe('Nešto nije u redu. Pokušajte ponovo.')
  })
})

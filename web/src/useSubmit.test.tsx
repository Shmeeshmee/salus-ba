import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { somethingWrong } from './messages'
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
    render(<Form action={action} />)

    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect((await screen.findByRole('alert')).textContent).toBe(somethingWrong)
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Save' }).disabled).toBe(false)
  })
})

import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Dialog } from './Dialog'

afterEach(cleanup)

describe('Dialog', () => {
  it('closes only the inner of two nested dialogs on Escape (wave 6, BG2)', () => {
    const outer = vi.fn()
    const inner = vi.fn()
    render(
      <Dialog label="Patients" onClose={outer}>
        <button type="button">Search</button>
        <Dialog label="Patient card" onClose={inner}>
          <button type="button">Close</button>
        </Dialog>
      </Dialog>,
    )
    fireEvent.keyDown(screen.getByRole('dialog', { name: 'Patient card' }), { key: 'Escape' })
    expect(inner).toHaveBeenCalledTimes(1)
    expect(outer).not.toHaveBeenCalled()

    fireEvent.keyDown(screen.getByRole('dialog', { name: 'Patients' }), { key: 'Escape' })
    expect(outer).toHaveBeenCalledTimes(1)
  })

  it('keeps Tab inside when its last button is disabled, as a receipt without its service (WAVE7 RD3)', () => {
    render(
      <Dialog label="New receipt" onClose={vi.fn()}>
        <input aria-label="Service" />
        <button type="button">Cancel</button>
        <button type="submit" disabled>
          Issue receipt
        </button>
      </Dialog>,
    )
    const cancel = screen.getByRole('button', { name: 'Cancel' })
    cancel.focus()
    fireEvent.keyDown(cancel, { key: 'Tab' })
    expect(document.activeElement).toBe(screen.getByLabelText('Service'))

    fireEvent.keyDown(screen.getByLabelText('Service'), { key: 'Tab', shiftKey: true })
    expect(document.activeElement).toBe(cancel)
  })

  it('takes the focus itself when it holds no control, so Escape still closes it (WAVE7 RD3)', () => {
    const close = vi.fn()
    render(
      <Dialog label="Add a doctor" onClose={close}>
        <p>No other doctor.</p>
      </Dialog>,
    )
    const dialog = screen.getByRole('dialog', { name: 'Add a doctor' })
    expect(document.activeElement).toBe(dialog)
    fireEvent.keyDown(dialog, { key: 'Escape' })
    expect(close).toHaveBeenCalledTimes(1)
  })
})

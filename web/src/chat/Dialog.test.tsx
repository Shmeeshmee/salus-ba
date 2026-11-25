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
})

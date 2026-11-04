import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { Field } from './ui'

afterEach(cleanup)

describe('Field', () => {
  it('keeps the label on its input when the caller passes an id', () => {
    render(<Field label="E-mail" id="email" hint="Your work address" />)

    const input = screen.getByLabelText('E-mail')
    expect(input.getAttribute('aria-describedby')).toBe(`${input.id}-hint`)
    expect(screen.getByText('Your work address').id).toBe(`${input.id}-hint`)
  })
})

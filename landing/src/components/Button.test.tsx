import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { Button } from './Button'

afterEach(cleanup)

describe('Button', () => {
  it('is a link when it has an href', () => {
    render(<Button variant="ink" href="#demo">Book a demo</Button>)

    expect(screen.getByRole('link', { name: 'Book a demo' }).getAttribute('href')).toBe('#demo')
  })

  it('is a button that does not submit when it has no href', () => {
    render(<Button variant="line">See the product</Button>)

    expect(screen.getByRole('button', { name: 'See the product' }).getAttribute('type')).toBe('button')
  })
})

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { highlights } from '../../content/highlights'
import { Highlights } from './Highlights'

afterEach(cleanup)

describe('Highlights', () => {
  it('lists every highlight once for assistive technology, though the marquee shows it twice', () => {
    render(<Highlights />)

    const region = screen.getByRole('region', { name: highlights.label })
    for (const item of highlights.items) {
      const copies = screen.getAllByText(item)
      expect(copies).toHaveLength(2)
      expect(copies.filter((copy) => copy.getAttribute('aria-hidden') !== 'true')).toHaveLength(1)
    }
    expect(region).toBeTruthy()
  })
})

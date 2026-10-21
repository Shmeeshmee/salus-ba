import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { footer } from '../../content/footer'
import { Footer } from './Footer'

afterEach(cleanup)

describe('Footer', () => {
  it('renders the links from content and the current year', () => {
    render(<Footer />)

    for (const link of footer.links) {
      expect(screen.getByRole('link', { name: link.label }).getAttribute('href')).toBe(link.href)
    }
    expect(screen.getByRole('contentinfo').textContent).toContain(`© ${new Date().getFullYear()} Salus`)
  })
})

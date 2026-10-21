import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { App } from './App'

afterEach(cleanup)

describe('App', () => {
  it('lets keyboard users skip the hero to the page content', () => {
    render(<App />)

    const skip = screen.getByRole('link', { name: 'Skip to content' })
    expect(skip.getAttribute('href')).toBe(`#${screen.getByRole('main').id}`)
  })

  it('puts the sections in page order', () => {
    render(<App />)

    const headings = screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent)
    expect(headings[0]).toBe('One platform. Two apps. No waiting room.')
    expect(headings.at(-1)).toBe('Bring your clinic online.')
    expect(screen.getByRole('contentinfo')).toBeTruthy()
  })
})

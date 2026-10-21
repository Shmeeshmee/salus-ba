import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { demo } from '../../content/demo'
import { Demo } from './Demo'

afterEach(cleanup)

describe('Demo', () => {
  it('renders the heading and the demo link from content', () => {
    render(<Demo />)

    expect(screen.getByRole('heading', { level: 2, name: demo.heading })).toBeTruthy()
    expect(screen.getByRole('link', { name: demo.cta.label }).getAttribute('href')).toBe(demo.cta.href)
  })
})

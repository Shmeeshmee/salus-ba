import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { standards } from '../../content/standards'
import { Standards } from './Standards'

afterEach(cleanup)

describe('Standards', () => {
  it('renders the heading, body and every tile from content', () => {
    render(<Standards />)

    expect(screen.getByRole('heading', { level: 2, name: standards.heading })).toBeTruthy()
    expect(screen.getByText(standards.body)).toBeTruthy()
    for (const tile of standards.tiles) {
      expect(screen.getByText(tile.title)).toBeTruthy()
      expect(screen.getByText(tile.detail)).toBeTruthy()
    }
  })
})

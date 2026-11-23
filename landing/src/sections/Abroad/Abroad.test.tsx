import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { abroad } from '../../content/abroad'
import { Abroad } from './Abroad'

afterEach(cleanup)

describe('Abroad', () => {
  it('renders the heading, the body and every point from content', () => {
    render(<Abroad />)

    expect(screen.getByRole('heading', { level: 2, name: abroad.heading })).toBeTruthy()
    expect(screen.getByText(abroad.body)).toBeTruthy()
    const points = within(screen.getByRole('list')).getAllByRole('listitem')
    expect(points).toHaveLength(abroad.points.length)
    abroad.points.forEach((point, i) => {
      expect(within(points[i]).getByText(point.text)).toBeTruthy()
    })
  })

  it('marks only what is not built yet as coming', () => {
    render(<Abroad />)

    const points = within(screen.getByRole('list')).getAllByRole('listitem')
    abroad.points.forEach((point, i) => {
      expect(within(points[i]).queryByText(abroad.comingLabel) !== null).toBe(point.coming)
    })
    expect(abroad.points.some((point) => point.coming)).toBe(true)
  })
})

import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { renderInLanguage } from '../i18n/renderInLanguage'
import { Heatmap } from './Heatmap'

afterEach(cleanup)

const cells = [
  { weekday: 1, hour: 9, count: 14 },
  { weekday: 1, hour: 10, count: 3 },
  { weekday: 7, hour: 3, count: null },
]

describe('Heatmap', () => {
  it('has a cell for every weekday and hour, a missing one being 0', () => {
    renderInLanguage(<Heatmap title="Busiest hours" cells={cells} />, 'en')
    const chart = screen.getByRole('figure', { name: 'Busiest hours' })
    expect(within(chart).getAllByRole('img')).toHaveLength(7 * 24)
    expect(within(chart).getByRole('img', { name: 'Mon 09:00–10:00: 14' })).toBeTruthy()
    expect(within(chart).getByRole('img', { name: 'Tue 09:00–10:00: 0' })).toBeTruthy()
    expect(within(chart).getByRole('img', { name: 'Sun 03:00–04:00: Hidden' })).toBeTruthy()
  })

  it('moves by hour with left and right and by day with up and down', () => {
    renderInLanguage(<Heatmap title="Busiest hours" cells={cells} />, 'en')
    const nine = screen.getByRole('img', { name: 'Mon 09:00–10:00: 14' })
    fireEvent.focus(nine)
    expect(screen.getByRole('tooltip').textContent).toContain('14')
    fireEvent.keyDown(nine, { key: 'ArrowRight' })
    expect(document.activeElement?.getAttribute('aria-label')).toBe('Mon 10:00–11:00: 3')
    fireEvent.keyDown(document.activeElement!, { key: 'ArrowDown' })
    expect(document.activeElement?.getAttribute('aria-label')).toBe('Tue 10:00–11:00: 0')
    fireEvent.keyDown(document.activeElement!, { key: 'Home' })
    expect(document.activeElement?.getAttribute('aria-label')).toBe('Tue 00:00–01:00: 0')
  })

  it('shows the hidden swatch in the legend only when a cell is hidden', () => {
    renderInLanguage(<Heatmap title="Busiest hours" cells={cells.slice(0, 2)} />, 'en')
    expect(within(screen.getByRole('list', { name: 'Legend' })).queryByText('Hidden')).toBeNull()
    cleanup()
    renderInLanguage(<Heatmap title="Busiest hours" cells={cells} />, 'en')
    expect(within(screen.getByRole('list', { name: 'Legend' })).getByText('Hidden')).toBeTruthy()
  })

  it('says so when every cell is empty', () => {
    renderInLanguage(<Heatmap title="Busiest hours" cells={[]} />, 'en')
    expect(screen.getByRole('figure', { name: 'Busiest hours' }).textContent).toContain('No data for these filters.')
  })
})

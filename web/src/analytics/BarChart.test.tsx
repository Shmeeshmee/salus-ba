import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderInLanguage } from '../i18n/renderInLanguage'
import { BarChart } from './BarChart'

afterEach(cleanup)

const bars = [
  { key: 'respiratory-infections', label: 'Respiratory infections', count: 31 },
  { key: 'skin', label: 'Skin conditions', count: 4 },
  { key: 'ear', label: 'Ear conditions', count: null },
]

describe('BarChart', () => {
  it('lists the bars largest first, a hidden one last, each named with its exact value', () => {
    renderInLanguage(<BarChart title="Top diagnosis groups" bars={bars} />, 'en')
    const chart = screen.getByRole('figure', { name: 'Top diagnosis groups' })
    expect(within(chart).getAllByRole('img').map((m) => m.getAttribute('aria-label'))).toEqual([
      'Respiratory infections: 31',
      'Skin conditions: 4',
      'Ear conditions: Hidden',
    ])
  })

  it('reads a hidden bar as Hidden, with the privacy tooltip', () => {
    renderInLanguage(<BarChart title="Regions" bars={bars} />, 'en')
    const hidden = screen.getByRole('img', { name: 'Ear conditions: Hidden' })
    expect(hidden.textContent).toContain('Hidden')
    fireEvent.mouseEnter(hidden)
    expect(screen.getByRole('tooltip').textContent).toBe('Ear conditionsHiddenSmall or dominated groups are hidden to protect privacy. A hidden group can hold more patients than a group shown.')
  })

  it('describes a hidden bar with the privacy explanation, for a screen reader', () => {
    renderInLanguage(<BarChart title="Regions" bars={bars} />, 'en')
    const hidden = screen.getByRole('img', { name: 'Ear conditions: Hidden' })
    expect(document.getElementById(hidden.getAttribute('aria-describedby') ?? '')?.textContent).toBe('Small or dominated groups are hidden to protect privacy. A hidden group can hold more patients than a group shown.')
    expect(screen.getByRole('img', { name: 'Skin conditions: 4' }).getAttribute('aria-describedby')).toBeNull()
  })

  it('leaves a bar that has no breakdown unopenable', () => {
    const onSelect = vi.fn()
    renderInLanguage(<BarChart title="Subgroups" bars={[...bars, { key: '', label: 'No subgroup', count: 2, fixed: true }]} onSelect={onSelect} />, 'en')
    const none = screen.getByRole('img', { name: 'No subgroup: 2' })
    fireEvent.click(none)
    fireEvent.keyDown(none, { key: 'Enter' })
    expect(onSelect).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Skin conditions: 4. Open its breakdown' })).toBeTruthy()
  })

  it('drills down with a click or the keyboard when a bar can be opened', () => {
    const onSelect = vi.fn()
    renderInLanguage(<BarChart title="Top diagnosis groups" bars={bars} onSelect={onSelect} />, 'en')
    const first = screen.getByRole('button', { name: 'Respiratory infections: 31. Open its breakdown' })
    fireEvent.click(first)
    expect(onSelect).toHaveBeenCalledWith('respiratory-infections')
    fireEvent.keyDown(first, { key: 'ArrowDown' })
    expect(document.activeElement?.getAttribute('aria-label')).toBe('Skin conditions: 4. Open its breakdown')
    fireEvent.keyDown(document.activeElement!, { key: 'Enter' })
    expect(onSelect).toHaveBeenLastCalledWith('skin')
  })

  it('labels the unit and a partial sum', () => {
    renderInLanguage(<BarChart title="Regions" bars={bars} unit="Patient-months" partial />, 'en')
    const chart = screen.getByRole('figure', { name: 'Regions' })
    const note = chart.querySelector('.chart-note') as HTMLElement
    expect(within(note).getByText('Patient-months')).toBeTruthy()
    expect(within(note).getByText('Partial')).toBeTruthy()
  })

  it('says so when every bar is zero', () => {
    renderInLanguage(<BarChart title="Top diagnosis groups" bars={[{ key: 'skin', label: 'Skin conditions', count: 0 }]} />, 'en')
    expect(screen.getByRole('figure', { name: 'Top diagnosis groups' }).textContent).toContain('No data for these filters.')
  })
})

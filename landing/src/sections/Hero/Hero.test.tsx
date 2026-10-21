import { act, cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { hero } from '../../content/hero'
import { specialties } from '../../content/specialties'
import { stubReducedMotion } from '../../test/matchMedia'
import { Hero } from './Hero'

// The intro runs for 2.2 s before the first 4 s interval starts.
const FIRST_CHANGE_MS = 2200 + 4000

const heroRegion = () => screen.getByRole('region', { name: hero.label })

// The live region holds every specialty name; only the one on show is exposed to assistive technology.
function shownSpecialty() {
  const names = within(screen.getByRole('status')).getAllByText(/\.$/)
  return names.filter((name) => name.getAttribute('aria-hidden') !== 'true').map((name) => name.textContent)
}

const dot = (name: string) => screen.getByRole('button', { name: `Show ${name.toLowerCase()}` })

const scrollIntoView = vi.fn()

beforeEach(() => {
  vi.useFakeTimers()
  scrollIntoView.mockClear()
  Element.prototype.scrollIntoView = scrollIntoView
})
afterEach(() => {
  cleanup()
  vi.useRealTimers()
})

describe('Hero', () => {
  it('shows the headline and the first specialty', () => {
    render(<Hero />)

    expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Your doctor, one call away.')
    expect(shownSpecialty()).toEqual(['Cardiology.'])
    expect(screen.getByText('01')).toBeTruthy()
    expect(dot('Cardiology').getAttribute('aria-pressed')).toBe('true')
  })

  it('shows the second specialty after the interval', () => {
    render(<Hero />)

    act(() => vi.advanceTimersByTime(FIRST_CHANGE_MS))

    expect(shownSpecialty()).toEqual(['Dermatology.'])
    expect(dot('Dermatology').getAttribute('aria-pressed')).toBe('true')
    expect(dot('Cardiology').getAttribute('aria-pressed')).toBe('false')
  })

  it('selects the specialty whose dot is clicked', () => {
    render(<Hero />)

    fireEvent.click(dot('Pediatrics'))

    expect(shownSpecialty()).toEqual(['Pediatrics.'])
    expect(screen.getByText('03')).toBeTruthy()
    expect(dot('Pediatrics').getAttribute('aria-pressed')).toBe('true')
    expect(dot('Cardiology').getAttribute('aria-pressed')).toBe('false')
  })

  it('recolours the hero with the selected specialty', () => {
    render(<Hero />)
    const [first, , , last] = specialties
    expect(heroRegion().style.getPropertyValue('--bg')).toBe(first.palette.bg)

    fireEvent.click(dot(last.name))

    expect(heroRegion().style.getPropertyValue('--bg')).toBe(last.palette.bg)
    expect(heroRegion().style.getPropertyValue('--ink')).toBe(last.palette.ink)
    expect(heroRegion().style.getPropertyValue('--acc')).toBe(last.palette.accent)
  })

  it('keeps the browser theme colour in step with the hero', () => {
    const meta = document.head.appendChild(document.createElement('meta'))
    meta.name = 'theme-color'
    render(<Hero />)

    fireEvent.click(dot('Pediatrics'))

    expect(meta.content).toBe(specialties[2].palette.bg)
    meta.remove()
  })

  it('stays on the first specialty when motion is reduced', () => {
    const media = stubReducedMotion(true)
    render(<Hero />)

    act(() => vi.advanceTimersByTime(FIRST_CHANGE_MS * 3))

    expect(shownSpecialty()).toEqual(['Cardiology.'])
    media.restore()
  })

  it('describes the phone and hides the decoration from assistive technology', () => {
    render(<Hero />)

    expect(screen.getByRole('img', { name: hero.phone.label })).toBeTruthy()
    expect(screen.queryByText(hero.monitor.lead)?.closest('[aria-hidden="true"]')).toBeTruthy()
    expect(screen.queryByText(hero.chips.booking.title)?.closest('[aria-hidden="true"]')).toBeTruthy()
  })

  it('glides to the page from the product button and the scroll cue', () => {
    render(
      <>
        <Hero />
        <main id="page" />
      </>,
    )

    fireEvent.click(screen.getByRole('button', { name: hero.secondaryCta }))
    act(() => vi.advanceTimersByTime(1100))
    fireEvent.click(screen.getByRole('button', { name: hero.scrollCue.ariaLabel }))

    expect(scrollIntoView).toHaveBeenCalledTimes(2)
  })

  it('links to the demo and the page sections', () => {
    render(<Hero />)

    const nav = screen.getByRole('navigation', { name: 'Main' })
    expect(within(nav).getByRole('link', { name: 'Pricing' }).getAttribute('href')).toBe('#pricing')
    expect(screen.getByRole('link', { name: 'Salus home' }).getAttribute('href')).toBe('#top')
    for (const demo of screen.getAllByRole('link', { name: /Book a demo/ })) {
      expect(demo.getAttribute('href')).toBe('#demo')
    }
  })
})

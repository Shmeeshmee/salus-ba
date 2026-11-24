import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ClinicSettingsNav } from './ClinicSettingsNav'
import { renderInLanguage } from './i18n/renderInLanguage'

afterEach(cleanup)

describe('ClinicSettingsNav', () => {
  it("links each section of the clinic settings page, one click away (wave 4 walk, finding 8)", () => {
    renderInLanguage(<ClinicSettingsNav />, 'en')
    const nav = within(screen.getByRole('navigation', { name: 'On this page' }))
    expect(nav.getAllByRole('link').map((l) => [l.textContent, l.getAttribute('href')])).toEqual([
      ['Clinic details', '#clinic-details'],
      ['Rescheduling', '#reschedule-settings'],
      ['Repeat no-shows', '#noshow-settings'],
      ['Reminders', '#clinic-reminders'],
      ['Clinic data export', '#clinic-export'],
      ['Import patients', '#clinic-import'],
      ['Waitlist', '#clinic-waitlist'],
      ['Profile for patients abroad', '#clinic-profile'],
    ])
  })

  it('scrolls to the section and moves the focus there, without changing the address', () => {
    const target = document.createElement('section')
    target.id = 'clinic-export'
    const scroll = vi.fn()
    target.scrollIntoView = scroll
    document.body.append(target)
    renderInLanguage(<ClinicSettingsNav />, 'en')
    fireEvent.click(screen.getByRole('link', { name: 'Clinic data export' }))
    expect(scroll).toHaveBeenCalled()
    expect(document.activeElement).toBe(target)
    expect(location.hash).toBe('')
    target.remove()
  })

  it('shows the links in Bosnian', () => {
    renderInLanguage(<ClinicSettingsNav />, 'bs')
    expect(screen.getByRole('navigation', { name: 'Na ovoj stranici' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Podaci o klinici' })).toBeTruthy()
  })
})

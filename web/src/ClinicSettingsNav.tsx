import type { MouseEvent } from 'react'
import { useT } from './i18n/useT'

// clinicSections are the sections of the clinic settings page (/clinic) in their order: the id each section carries and
// the key of its title. Waitlist and the profile for patients abroad follow ClinicSettings on the page (App).
const clinicSections: readonly { id: string; label: string }[] = [
  { id: 'clinic-details', label: 'admin.settingsNavDetails' },
  { id: 'reschedule-settings', label: 'appointments.reschedule' },
  { id: 'noshow-settings', label: 'calendar.noshow.settingsTitle' },
  { id: 'clinic-reminders', label: 'reminders.title' },
  { id: 'clinic-export', label: 'admin.exportTitle' },
  { id: 'clinic-import', label: 'patientImport.title' },
  { id: 'clinic-waitlist', label: 'waitlist.settingsTitle' },
  { id: 'clinic-profile', label: 'clinicProfile.title' },
]

// ClinicSettingsNav is the clinic settings page's in-page navigation (wave 4 walk, finding 8): each section one click
// away on a page that grew long. A link scrolls to its section and moves the focus there, so a keyboard or screen
// reader user lands where a sighted one does; the address stays as it is (the app's routes are paths).
export function ClinicSettingsNav() {
  const t = useT()
  function go(e: MouseEvent, id: string) {
    const target = document.getElementById(id)
    if (!target) return
    e.preventDefault()
    target.scrollIntoView({ block: 'start' })
    if (!target.hasAttribute('tabindex')) target.setAttribute('tabindex', '-1')
    target.focus({ preventScroll: true })
  }
  return (
    <nav className="settings-nav" aria-label={t('admin.settingsNav')}>
      <ul>
        {clinicSections.map((s) => (
          <li key={s.id}>
            <a href={`#${s.id}`} onClick={(e) => go(e, s.id)}>
              {t(s.label)}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  )
}

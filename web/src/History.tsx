import { useState } from 'react'
import { binaryContentURL, type ApiClient } from './api/client'
import { participant, type Appointment, type FhirClient } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { Alert } from './ui'

type Item = components['schemas']['HistoryItem']
type Summary = components['schemas']['HistorySummary']
type T = ReturnType<typeof useT>

type Props = { api: ApiClient; fhir: FhirClient; patientId: string; appointmentId: string; zone: string }

// A Composition as its FHIR read gives it: the sections the history shows of a signed document.
type Composition = { section?: { title: string; text?: { div: string } }[] }

// PatientHistory is the practitioner's view of the patient's history (SAL-40, ADR 0014), from an appointment of
// theirs: the pre-visit summary on top (allergies, current medications, the last visit, what is open for the
// appointment), then the visits they may see and what belongs to each, newest first, one page at a time. A visit
// of another clinic is there only because the patient shared it, and says so with the clinic's name. The server
// decides what is shown and audits every view; nothing here filters or counts.
export function PatientHistory({ api, fhir, patientId, appointmentId, zone }: Props) {
  const t = useT()
  const { state, set } = useLoad(async () => {
    const path = { params: { path: { id: patientId } } }
    const [summary, page] = await Promise.all([
      attempt(api.GET('/patients/{id}/history/summary', { params: { path: { id: patientId }, query: { appointment: appointmentId } } })),
      attempt(api.GET('/patients/{id}/history', path)),
    ])
    if (!summary.ok || !page.ok) throw new Error('history')
    return { summary: summary.data, items: page.data.items, next: page.data.next }
  }, [api, patientId, appointmentId])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  async function more() {
    if (state.name !== 'loaded' || !state.value.next) return
    setBusy(true)
    setError(undefined)
    const res = await attempt(api.GET('/patients/{id}/history', { params: { path: { id: patientId }, query: { cursor: state.value.next } } }))
    setBusy(false)
    if (!res.ok) {
      setError(t('history.loadFailed'))
      return
    }
    set({ ...state.value, items: [...state.value.items, ...res.data.items], next: res.data.next })
  }

  return (
    <section className="history" aria-label={t('history.title')}>
      <h2>{t('history.title')}</h2>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('history.loadFailed')} />}
      {state.name === 'loaded' && (
        <>
          <SummaryView summary={state.value.summary} zone={zone} />
          <h3>{t('history.items')}</h3>
          {state.value.items.length === 0 && <p className="muted">{t('history.empty')}</p>}
          <ul className="history-items">
            {state.value.items.map((it) => (
              <li key={it.reference}>
                <ItemRow fhir={fhir} item={it} zone={zone} />
              </li>
            ))}
          </ul>
          <Alert message={error} />
          {state.value.next && (
            <div className="actions">
              <button type="button" disabled={busy} onClick={more}>
                {t('history.more')}
              </button>
            </div>
          )}
        </>
      )}
    </section>
  )
}

function SummaryView({ summary, zone }: { summary: Summary; zone: string }) {
  const t = useT()
  return (
    <div className="history-summary">
      <h3>{t('history.summary')}</h3>
      <h4>{t('history.allergies')}</h4>
      {summary.allergies.length === 0 ? (
        <p className="muted">{t('history.noAllergies')}</p>
      ) : (
        <ul>
          {summary.allergies.map((a) => (
            <li key={a.reference}>
              <span className="history-name">{a.text}</span>
              {a.reaction && <span>{a.severity ? t('history.reactionSeverity', { reaction: a.reaction, severity: t(`history.severity.${a.severity}`) }) : a.reaction}</span>}
              {a.confirmed && <span className="tag">{t('history.confirmed')}</span>}
            </li>
          ))}
        </ul>
      )}
      <h4>{t('history.medications')}</h4>
      {summary.medications.length === 0 ? (
        <p className="muted">{t('history.noMedications')}</p>
      ) : (
        <ul>
          {summary.medications.map((m) => (
            <li key={m.reference}>
              <span className="history-name">{m.text}</span>
              {m.confirmed && <span className="tag">{t('history.confirmed')}</span>}
            </li>
          ))}
        </ul>
      )}
      <h4>{t('history.lastVisit')}</h4>
      {summary.lastVisit ? <ItemLine item={summary.lastVisit} zone={zone} /> : <p className="muted">{t('history.noLastVisit')}</p>}
      <h4>{t('history.openItems')}</h4>
      {summary.openItems.length === 0 ? (
        <p className="muted">{t('history.noOpenItems')}</p>
      ) : (
        <ul>
          {summary.openItems.map((o) => (
            <li key={o.reference}>{t(`history.open.${o.type}`)}</li>
          ))}
        </ul>
      )}
    </div>
  )
}

// ItemLine is an item's date, what it is, its title and where it comes from.
function ItemLine({ item, zone }: { item: Item; zone: string }) {
  const t = useT()
  const f = useFormat()
  return (
    <span className="history-line">
      <span className="history-date">{f.dateTime(new Date(item.date), zone)}</span>
      <span className="history-name">{item.title || t(`history.type.${item.type}`)}</span>
      {item.title && <span className="muted">{t(`history.type.${item.type}`)}</span>}
      {item.shared ? <span className="tag shared">{t('history.sharedFrom', { clinic: item.clinic.name })}</span> : <span className="muted">{item.clinic.name}</span>}
      {item.diagnoses && item.diagnoses.length > 0 && <span>{t('history.diagnoses', { codes: item.diagnoses.join(', ') })}</span>}
    </span>
  )
}

// CallHistory is PatientHistory on the call page, for the appointment's patient, in the practitioner's own time zone.
export function CallHistory({ api, fhir, appointmentId }: { api: ApiClient; fhir: FhirClient; appointmentId: string }) {
  const t = useT()
  const { state } = useLoad(async () => participant(await fhir.get<Appointment>(`Appointment/${encodeURIComponent(appointmentId)}`), 'Patient')?.id, [fhir, appointmentId])
  if (state.name === 'loading') return <p className="muted">{t('common.loading')}</p>
  if (state.name === 'failed' || !state.value) return <Alert message={t('history.loadFailed')} />
  return <PatientHistory api={api} fhir={fhir} patientId={state.value} appointmentId={appointmentId} zone={localZone} />
}

const localZone = Intl.DateTimeFormat().resolvedOptions().timeZone

// ItemRow is an item with what opens it: a signed document's sections in place, a file's download.
function ItemRow({ fhir, item, zone }: { fhir: FhirClient; item: Item; zone: string }) {
  const t = useT()
  const [open, setOpen] = useState<Composition>()
  const [failed, setFailed] = useState(false)
  const [busy, setBusy] = useState(false)
  const [, id] = item.reference.split('/')

  async function toggle() {
    if (open) {
      setOpen(undefined)
      return
    }
    setBusy(true)
    setFailed(false)
    try {
      setOpen(await fhir.get<Composition>(`Composition/${encodeURIComponent(id)}`))
    } catch {
      setFailed(true)
    }
    setBusy(false)
  }

  return (
    <>
      <ItemLine item={item} zone={zone} />
      {item.type === 'document' && (
        <button type="button" className="quiet" aria-expanded={open !== undefined} disabled={busy} onClick={toggle}>
          {t(open ? 'history.closeDocument' : 'history.openDocument')}
        </button>
      )}
      {item.type === 'attachment' && (
        <a className="quiet" href={binaryContentURL(id)} download={item.title || ''}>
          {t('attachments.download')}
        </a>
      )}
      {failed && <Alert message={t('history.documentFailed')} />}
      {open && <Sections composition={open} t={t} />}
    </>
  )
}

// Sections is a signed document's sections as plain text: the narrative's markup is not rendered.
function Sections({ composition, t }: { composition: Composition; t: T }) {
  return (
    <dl className="history-document" aria-label={t('history.type.document')}>
      {(composition.section ?? []).map((s, i) => (
        // A document's sections have no other identity than their place.
        <div key={i}>
          <dt>{s.title}</dt>
          <dd>{s.text ? plain(s.text.div) : '-'}</dd>
        </div>
      ))}
    </dl>
  )
}

// plain is the narrative's text, a line per paragraph or list item.
function plain(xhtml: string): string {
  const body = new DOMParser().parseFromString(xhtml, 'text/html').body
  const lines = Array.from(body.querySelectorAll('p, li'), (e) => e.textContent?.trim() ?? '')
  return lines.length > 0 ? lines.join('\n') : (body.textContent?.trim() ?? '')
}

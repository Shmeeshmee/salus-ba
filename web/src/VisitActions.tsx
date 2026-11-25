import { useState } from 'react'
import type { ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import { AppointmentChanges } from './AppointmentChanges'
import { CarePlans } from './CarePlans'
import { PatientHistory } from './History'
import { useT } from './i18n/useT'
import { useLoad } from './load'
import { QuoteDialog, VisitQuotes } from './QuoteDialog'
import { ReceiptDialog } from './ReceiptDialog'
import { receiptPdfURL } from './receipts'
import { Notice } from './ui'
import { mergeReceipts, type VisitReceipt } from './visitReceipts'

// VisitReceipts lists a visit's receipts, each with its PDF and a reversed one marked (docs/clinic-cms/walk/wave3,
// findings 1 and 2).
export function VisitReceipts({ receipts }: { receipts: VisitReceipt[] }) {
  const t = useT()
  if (receipts.length === 0) return null
  return (
    <p className="visit-receipts">
      {receipts.map((r) => (
        <span key={r.id} className="visit-receipt">
          <span>{t('receipts.visitReceipt', { number: r.number })}</span>
          {r.reversed && <span className="tag">{t('receipts.reversed')}</span>}
          <a href={receiptPdfURL(r.id)} target="_blank" rel="noreferrer">
            {t('receipts.pdf')}
          </a>
        </span>
      ))}
    </p>
  )
}

type Props = {
  api: ApiClient
  fhir: FhirClient
  appointmentId: string
  patientId?: string
  serviceId?: string
  zone: string
  history: boolean // the patient's history, for the visit's own practitioner
  receipts: boolean // issuing a receipt, for the admin and the visit's practitioner
  quotes: boolean // writing and showing quotes, as receipts
  admin: boolean // the clinic's admin reads the visit's receipts
  plans?: boolean // the patient's treatment plans (TP2), for the admin and the visit's practitioner
  practitionerId?: string // the signed-in practitioner, whose times a plan's booking offers first
}

// VisitActions are the visit page's secondary actions in one row of one style, each opening below it: the patient's
// history, the visit's changes, its receipt, its quotes and the patient's treatment plans (docs/clinic-cms/walk/wave3,
// finding 4; TP2). Opening the page
// reads (and audits) none of them but the admin's list of the visit's receipts.
export function VisitActions({ api, fhir, appointmentId, patientId, serviceId, zone, history, receipts, quotes, admin, plans = false, practitionerId }: Props) {
  const t = useT()
  const [historyOpen, setHistoryOpen] = useState(false)
  const [changesOpen, setChangesOpen] = useState(false)
  const [quotesShown, setQuotesShown] = useState(false)
  const [plansShown, setPlansShown] = useState(false)
  const [dialog, setDialog] = useState<'receipt' | 'quote'>()
  const [sent, setSent] = useState(0) // reloads the shown quotes after one is sent
  const [notice, setNotice] = useState<string>()
  const [issued, setIssued] = useState<VisitReceipt[]>([])
  const { state } = useLoad(async (): Promise<VisitReceipt[]> => {
    if (!admin || !receipts) return []
    const res = await attempt(api.GET('/clinic/receipts', { params: { query: { appointmentId } } }))
    return res.ok ? res.data.map((r) => ({ id: r.id, number: r.number, reversed: !!r.reversedById })) : []
  }, [api, appointmentId, admin, receipts])
  const all = mergeReceipts(state.name === 'loaded' ? state.value : [], issued)
  const active = all.filter((r) => !r.reversed)
  return (
    <div className="visit-actions">
      <div className="actions">
        {history && (
          <button type="button" aria-expanded={historyOpen} onClick={() => setHistoryOpen(!historyOpen)}>
            {t(historyOpen ? 'history.hideHistory' : 'history.showHistory')}
          </button>
        )}
        <button type="button" aria-expanded={changesOpen} onClick={() => setChangesOpen(!changesOpen)}>
          {t(changesOpen ? 'appointments.hideChanges' : 'appointments.showChanges')}
        </button>
        {receipts && (
          <button type="button" onClick={() => setDialog('receipt')}>
            {t(active.length > 0 ? 'receipts.issueAnother' : 'receipts.dialog.issue')}
          </button>
        )}
        {quotes && (
          <>
            <button type="button" onClick={() => setDialog('quote')}>
              {t('receipts.quote.write')}
            </button>
            <button type="button" aria-expanded={quotesShown} onClick={() => setQuotesShown(!quotesShown)}>
              {t(quotesShown ? 'receipts.quote.hide' : 'receipts.quote.show')}
            </button>
          </>
        )}
        {plans && (
          <button type="button" aria-expanded={plansShown} onClick={() => setPlansShown(!plansShown)}>
            {t(plansShown ? 'carePlans.hide' : 'carePlans.show')}
          </button>
        )}
      </div>
      <Notice message={notice} />
      <VisitReceipts receipts={all} />
      {historyOpen && patientId && <PatientHistory api={api} fhir={fhir} patientId={patientId} appointmentId={appointmentId} zone={zone} />}
      {changesOpen && <AppointmentChanges api={api} id={appointmentId} zone={zone} />}
      {quotesShown && <VisitQuotes key={sent} api={api} appointmentId={appointmentId} zone={zone} />}
      {plansShown && patientId && <CarePlans api={api} fhir={fhir} patientId={patientId} appointmentId={appointmentId} practitionerId={practitionerId} zone={zone} />}
      {dialog === 'receipt' && (
        <ReceiptDialog
          api={api}
          appointmentId={appointmentId}
          serviceId={serviceId}
          issued={active.map((r) => r.number)}
          onClose={() => setDialog(undefined)}
          onIssued={(r) => {
            setDialog(undefined)
            setIssued((all) => [...all, { id: r.id, number: r.number, reversed: false }])
            setNotice(t('receipts.issued', { number: r.number }))
          }}
        />
      )}
      {dialog === 'quote' && patientId && (
        <QuoteDialog
          api={api}
          appointmentId={appointmentId}
          patientId={patientId}
          serviceId={serviceId}
          onClose={() => setDialog(undefined)}
          onSent={() => {
            setDialog(undefined)
            setSent((n) => n + 1)
            setNotice(t('receipts.quote.sent'))
          }}
        />
      )}
    </div>
  )
}

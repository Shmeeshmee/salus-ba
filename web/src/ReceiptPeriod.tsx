import { Field } from './ui'
import { useT } from './i18n/useT'
import type { Period } from './receipts'

// ReceiptPeriod is the two dates the receipts list and the report are read for.
export function ReceiptPeriod({ period, onChange }: { period: Period; onChange: (p: Period) => void }) {
  const t = useT()
  return (
    <>
      <Field label={t('receipts.from')} type="date" value={period.from} max={period.to} onChange={(e) => onChange({ ...period, from: e.target.value })} />
      <Field label={t('receipts.to')} type="date" value={period.to} min={period.from} onChange={(e) => onChange({ ...period, to: e.target.value })} />
    </>
  )
}

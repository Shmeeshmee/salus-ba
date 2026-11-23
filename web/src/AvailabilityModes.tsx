import { useT } from './i18n/useT'
import { visitModes, type VisitMode } from './visitMode'

// ModeSelect picks how patients attend the slots of one row of working hours: by video or at the clinic (ADR 0016).
// A practitioner who does both has a row for each; the API refuses rows that overlap.
export function ModeSelect({ label, value, onChange }: { label: string; value: VisitMode; onChange: (mode: VisitMode) => void }) {
  const t = useT()
  return (
    <select aria-label={label} value={value} onChange={(e) => onChange(e.target.value === 'in-person' ? 'in-person' : 'video')}>
      {visitModes.map((m) => (
        <option key={m} value={m}>
          {t(modeLabel[m])}
        </option>
      ))}
    </select>
  )
}

// modeLabel is the catalog key of each mode's name.
const modeLabel = { video: 'calendar.mode.video', 'in-person': 'calendar.mode.inPerson' } as const

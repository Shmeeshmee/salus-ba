import { useT } from './i18n/useT'
import type { Member } from './staff'

// PractitionerPicker chooses whose calendar or hours an admin works on.
export function PractitionerPicker({ staff, value, onChange }: { staff: Member[]; value: string; onChange: (practitionerId: string) => void }) {
  const t = useT()
  return (
    <div className="field picker">
      <label htmlFor="practitioner-picker">{t('common.practitioner')}</label>
      <select id="practitioner-picker" value={value} onChange={(e) => onChange(e.target.value)}>
        {staff.map((m) => (
          <option key={m.practitionerId} value={m.practitionerId}>
            {m.name}
          </option>
        ))}
      </select>
    </div>
  )
}

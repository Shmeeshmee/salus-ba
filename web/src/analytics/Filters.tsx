import { useId } from 'react'
import { useFormat, useT } from '../i18n/useT'
import type { DiagnosisGroup, Member, Month, Place, Specialty } from './data'
import type { Diagnosis, Geo } from './drill'
import { usePlaceName, useSpecialtyName } from './names'

// The analytics filters (plan §9 ANA-4): a month range, the diagnosis group with its subgroup, the place (entity
// or region, then city), the specialty and the practitioner. Each is a plain select; "" is every value.

type Option = { value: string; label: string }

export function Choice({ label, value, options, all, onChange }: { label: string; value: string; options: Option[]; all?: string; onChange: (value: string) => void }) {
  const id = useId()
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <select id={id} value={value} onChange={(e) => onChange(e.target.value)}>
        {all !== undefined && <option value="">{all}</option>}
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </div>
  )
}

export function MonthRange({ months, from, to, onChange }: { months: Month[]; from: Month; to: Month; onChange: (range: { from: Month; to: Month }) => void }) {
  const t = useT()
  const { month } = useFormat()
  const options = months.map((m) => ({ value: m, label: month(m, 'long') }))
  return (
    <>
      <Choice label={t('analytics.from')} value={from} options={options} onChange={(f) => onChange({ from: f, to: f > to ? f : to })} />
      <Choice label={t('analytics.to')} value={to} options={options} onChange={(v) => onChange({ from: v < from ? v : from, to: v })} />
    </>
  )
}

export function DiagnosisFilter({ groups, value, onChange }: { groups: DiagnosisGroup[]; value: Diagnosis; onChange: (value: Diagnosis) => void }) {
  const t = useT()
  const top = groups.filter((g) => !g.parent).map((g) => ({ value: g.code, label: g.name }))
  const subgroups = groups.filter((g) => value.group && g.parent === value.group).map((g) => ({ value: g.code, label: g.name }))
  return (
    <>
      <Choice label={t('analytics.group')} value={value.group ?? ''} options={top} all={t('analytics.allGroups')} onChange={(g) => onChange(g ? { group: g } : {})} />
      {subgroups.length > 0 && (
        <Choice
          label={t('analytics.subgroup')}
          value={value.subgroup ?? ''}
          options={subgroups}
          all={t('analytics.wholeGroup')}
          onChange={(s) => onChange(s ? { group: value.group, subgroup: s } : { group: value.group })}
        />
      )}
    </>
  )
}

export function PlaceFilter({ places, value, onChange, cities = true }: { places: Place[]; value: Geo; onChange: (value: Geo) => void; cities?: boolean }) {
  const t = useT()
  const name = usePlaceName(places)
  const options = places.filter((p) => p.kind !== 'city').map((p) => ({ value: p.id, label: name(p.id) }))
  const region = places.find((p) => p.id === value.place && p.kind === 'region')
  const cityOptions = places.filter((p) => region && p.kind === 'city' && p.parent === region.id).map((p) => ({ value: p.id, label: name(p.id) }))
  return (
    <>
      <Choice label={t('analytics.region')} value={value.place ?? ''} options={options} all={t('analytics.allRegions')} onChange={(p) => onChange(p ? { place: p } : {})} />
      {cities && cityOptions.length > 0 && (
        <Choice
          label={t('analytics.city')}
          value={value.city ?? ''}
          options={cityOptions}
          all={t('analytics.allCities')}
          onChange={(c) => onChange(c ? { place: value.place, city: c } : { place: value.place })}
        />
      )}
    </>
  )
}

export function SpecialtyFilter({ specialties, value, onChange }: { specialties: Specialty[]; value: string; onChange: (value: string) => void }) {
  const t = useT()
  const name = useSpecialtyName(specialties)
  return <Choice label={t('analytics.specialty')} value={value} options={specialties.map((s) => ({ value: s.code, label: name(s.code) }))} all={t('analytics.allSpecialties')} onChange={onChange} />
}

export function PractitionerFilter({ staff, value, onChange }: { staff: Member[]; value: string; onChange: (value: string) => void }) {
  const t = useT()
  return <Choice label={t('analytics.practitioner')} value={value} options={staff.map((m) => ({ value: m.roleId, label: m.name }))} all={t('analytics.allPractitioners')} onChange={onChange} />
}

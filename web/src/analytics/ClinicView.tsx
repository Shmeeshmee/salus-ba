import { useState } from 'react'
import { useT } from '../i18n/useT'
import { useLoad } from '../load'
import { BarChart } from './BarChart'
import { Loaded } from './chart'
import { monthsBetween, type AnalyticsData, type ClinicQuery, type DiagnosisGroup, type Member, type Month, type Specialty } from './data'
import { drillInto, drillOf, type Diagnosis } from './drill'
import { DiagnosisFilter, MonthRange, PractitionerFilter, SpecialtyFilter } from './Filters'
import { Heatmap } from './Heatmap'
import { LineChart } from './LineChart'

export type ClinicReference = { groups: DiagnosisGroup[]; specialties: Specialty[]; staff: Member[] }

const statuses = ['booked', 'fulfilled', 'noshow', 'cancelled'] as const

// monthsUpTo is the 36 months up to the month of now, the clinic's range (plan §4: 3 years of statistics).
function monthsUpTo(now: Date): Month[] {
  const to = now.toISOString().slice(0, 7)
  const start = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - 35, 1)).toISOString().slice(0, 7)
  return monthsBetween(start, to)
}

// ClinicView is My clinic (SAL-20, plan §4): the clinic's appointments over time and by weekday and hour, its
// diagnoses by group with the drill-down, and for an admin the breakdown by practitioner. A practitioner who is not
// an admin gets their own role's diagnoses from the API (rule C2); the view says so and offers neither a practitioner
// nor a specialty, which the API refuses them.
export function ClinicView({ data, reference, admin, now }: { data: AnalyticsData; reference: ClinicReference; admin: boolean; now?: Date }) {
  const t = useT()
  const [months] = useState(() => monthsUpTo(now ?? new Date())) // now: a test's fixed clock
  const [range, setRange] = useState({ from: months[months.length - 12], to: months[months.length - 1] })
  const [specialty, setSpecialty] = useState('')
  const [role, setRole] = useState('')
  const [diagnosis, setDiagnosis] = useState<Diagnosis>({})
  const by = drillOf(diagnosis, reference.groups)

  const query: ClinicQuery = { ...range, ...(specialty ? { specialty } : {}), ...(role ? { practitionerRole: role } : {}) }
  const deps = [data, range.from, range.to, specialty, role]
  const appointments = useLoad(() => data.clinicAppointments(query), deps)
  const diagnosisDeps = [...deps, diagnosis.group, diagnosis.subgroup]
  const bars = useLoad(() => data.clinicDiagnoses({ ...query, ...diagnosis, by }), [...diagnosisDeps, by])
  const perMonth = useLoad(() => data.clinicDiagnoses({ ...query, ...diagnosis, by: 'month' }), diagnosisDeps)

  const groupName = (code: string) => reference.groups.find((g) => g.code === code)?.name ?? code
  const barsTitle = by === 'group' ? t('analytics.topGroups') : by === 'subgroup' ? t('analytics.subgroupsOf', { name: groupName(diagnosis.group!) }) : t('analytics.codesIn', { name: groupName(diagnosis.subgroup ?? diagnosis.group!) })
  const staffName = (roleId: string) => reference.staff.find((m) => m.roleId === roleId)?.name ?? roleId
  const periods = monthsBetween(range.from, range.to)

  return (
    <div className="analytics-view">
      <div className="analytics-filters" role="group" aria-label={t('analytics.filters')}>
        <MonthRange months={months} from={range.from} to={range.to} onChange={setRange} />
        {admin && <SpecialtyFilter specialties={reference.specialties} value={specialty} onChange={setSpecialty} />}
        {admin && <PractitionerFilter staff={reference.staff} value={role} onChange={setRole} />}
        <DiagnosisFilter groups={reference.groups} value={diagnosis} onChange={setDiagnosis} />
      </div>
      {!admin && <p className="muted small">{t('analytics.ownDiagnoses')}</p>}
      <div className="analytics-grid">
        <Loaded state={appointments.state}>
          {(a) => (
            <LineChart
              title={t('analytics.appointments')}
              periods={a.series.map((s) => s.period)}
              series={statuses.map((key, i) => ({ key, label: t(`analytics.${key}`), color: `var(--series-${i + 1})`, values: a.series.map((s) => s[key]) }))}
            />
          )}
        </Loaded>
        <Loaded state={appointments.state}>{(a) => <Heatmap title={t('analytics.busiest')} cells={a.heatmap} />}</Loaded>
        <div className="analytics-wide">
          <DrillPath diagnosis={diagnosis} name={groupName} onChange={setDiagnosis} />
          <Loaded state={bars.state}>
            {(d) => (
              <BarChart
                title={barsTitle}
                bars={d.rows.map((r) => ({ key: r.key, label: by === 'code' ? r.key : groupName(r.key), count: r.count }))}
                unit={t('analytics.diagnoses')}
                onSelect={by === 'code' ? undefined : (key) => setDiagnosis(drillInto(diagnosis, by, key))}
              />
            )}
          </Loaded>
        </div>
        <Loaded state={perMonth.state}>
          {(d) => (
            <LineChart
              title={t('analytics.diagnosesPerMonth')}
              periods={periods}
              series={[{ key: 'diagnoses', label: t('analytics.diagnoses'), color: 'var(--series-2)', values: periods.map((p) => d.rows.find((r) => r.key === p)?.count ?? 0) }]}
            />
          )}
        </Loaded>
        {admin && (
          <Loaded state={appointments.state}>
            {(a) => (
              <BarChart
                title={t('analytics.byPractitioner')}
                bars={(a.practitioners ?? []).map((p) => ({ key: p.practitionerRole, label: staffName(p.practitionerRole), count: p.fulfilled }))}
              />
            )}
          </Loaded>
        )}
      </div>
    </div>
  )
}

// DrillPath is the way back up the diagnosis drill-down: every level above the bars shown.
export function DrillPath({ diagnosis, name, onChange }: { diagnosis: Diagnosis; name: (code: string) => string; onChange: (d: Diagnosis) => void }) {
  const t = useT()
  if (!diagnosis.group) return null
  return (
    <nav className="drill-path" aria-label={t('analytics.drillPath')}>
      <button type="button" className="link" onClick={() => onChange({})}>
        {t('analytics.allGroups')}
      </button>
      <span aria-hidden="true">›</span>
      {diagnosis.subgroup ? (
        <>
          <button type="button" className="link" onClick={() => onChange({ group: diagnosis.group })}>
            {name(diagnosis.group)}
          </button>
          <span aria-hidden="true">›</span>
          <span aria-current="page">{name(diagnosis.subgroup)}</span>
        </>
      ) : (
        <span aria-current="page">{name(diagnosis.group)}</span>
      )}
    </nav>
  )
}

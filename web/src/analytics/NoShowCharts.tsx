import { useT } from '../i18n/useT'
import { BarChart, type Bar } from './BarChart'
import { noShowRate, type ClinicAppointments, type Month, type Specialty } from './data'
import { LineChart } from './LineChart'
import { useSpecialtyName } from './names'

// rated is the bar of a key's no-show rate, or none when it closed no appointment: no bar is not a 0 % bar.
function rated(key: string, label: string, noshow: number, fulfilled: number): Bar[] {
  const count = noShowRate(noshow, fulfilled)
  return count === undefined ? [] : [{ key, label, count, fixed: true }]
}

type Props = {
  figures: ClinicAppointments
  periods: Month[]
  admin: boolean
  specialties: Specialty[]
  staffName: (roleId: string) => string
}

// NoShowCharts are My clinic's no-show reports (SAL-36): the no-show rate per month and the slots refilled from the
// waitlist; for an admin the rate per practitioner and per specialty (the API gives those to admins only, rule C2).
export function NoShowCharts({ figures, periods, admin, specialties, staffName }: Props) {
  const t = useT()
  const specialtyName = useSpecialtyName(specialties)
  const month = (p: Month) => figures.series.find((s) => s.period === p)
  return (
    <>
      <LineChart
        title={t('analytics.noShowRate')}
        periods={periods}
        series={[
          {
            key: 'rate',
            label: t('analytics.noShowRatePercent'),
            color: 'var(--series-3)',
            values: periods.map((p) => noShowRate(month(p)?.noshow ?? 0, month(p)?.fulfilled ?? 0)),
            missing: t('analytics.noClosedAppointments'),
          },
        ]}
      />
      <LineChart
        title={t('analytics.refilled')}
        periods={periods}
        series={[{ key: 'refilled', label: t('analytics.refilledSlots'), color: 'var(--series-2)', values: periods.map((p) => month(p)?.refilled ?? 0) }]}
      />
      {admin && (
        <BarChart
          title={t('analytics.noShowRateByPractitioner')}
          unit={t('analytics.percent')}
          bars={(figures.practitioners ?? []).flatMap((p) => rated(p.practitionerRole, staffName(p.practitionerRole), p.noshow, p.fulfilled))}
        />
      )}
      {admin && (
        <BarChart
          title={t('analytics.noShowRateBySpecialty')}
          unit={t('analytics.percent')}
          bars={(figures.specialties ?? []).flatMap((x) => rated(x.specialty, x.specialty === '' ? t('analytics.noSpecialty') : specialtyName(x.specialty), x.noshow, x.fulfilled))}
        />
      )}
    </>
  )
}

import { useT } from '../i18n/useT'
import { BarChart } from './BarChart'
import { noShowRate, type ClinicAppointments, type Month, type Specialty } from './data'
import { LineChart } from './LineChart'
import { useSpecialtyName } from './names'

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
        series={[{ key: 'rate', label: t('analytics.noShowRatePercent'), color: 'var(--series-3)', values: periods.map((p) => noShowRate(month(p)?.noshow ?? 0, month(p)?.fulfilled ?? 0)) }]}
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
          bars={(figures.practitioners ?? []).map((p) => ({ key: p.practitionerRole, label: staffName(p.practitionerRole), count: noShowRate(p.noshow, p.fulfilled), fixed: true }))}
        />
      )}
      {admin && (
        <BarChart
          title={t('analytics.noShowRateBySpecialty')}
          unit={t('analytics.percent')}
          bars={(figures.specialties ?? []).map((s) => ({
            key: s.specialty,
            label: s.specialty === '' ? t('analytics.noSpecialty') : specialtyName(s.specialty),
            count: noShowRate(s.noshow, s.fulfilled),
            fixed: true,
          }))}
        />
      )}
    </>
  )
}

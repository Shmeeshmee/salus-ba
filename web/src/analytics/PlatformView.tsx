import { useState } from 'react'
import { useT } from '../i18n/useT'
import { useLoad } from '../load'
import { TipTag } from '../ui'
import { BarChart, type Bar } from './BarChart'
import { Loaded } from './chart'
import { diagnosisBar, DrillPath, type ClinicReference } from './ClinicView'
import { dimensionsOf, type AnalyticsData, type Dimension, type Month, type Place, type PlatformCells, type PlatformTable, type Row } from './data'
import { drillInto, drillOf, geoFilter, type Diagnosis, type Geo } from './drill'
import { Choice, DiagnosisFilter, MonthRange, PlaceFilter, SpecialtyFilter } from './Filters'
import { Heatmap } from './Heatmap'
import { LineChart } from './LineChart'
import { usePlaceName, useSpecialtyName } from './names'

export type PlatformReference = ClinicReference & { places: Place[] }

const tables: PlatformTable[] = ['gd', 'gs', 'sd']

// named is a breakdown's row as a bar named by name.
const named = (name: (key: string) => string) => (r: Row): Bar => ({ key: r.key, label: name(r.key), count: r.count })

// PlatformView is All of Salus (SAL-21, plan §5 and §6): one table at a time, its two dimensions as the only
// filters, so the view never asks for a combination the API refuses. Counts are distinct patients of closed months;
// a sum over months is patient-months, partial when a cell is hidden. Rises are flagged without a count.
export function PlatformView({ data, reference, months, quarters }: { data: AnalyticsData; reference: PlatformReference; months: Month[]; quarters: string[] }) {
  const t = useT()
  const placeName = usePlaceName(reference.places)
  const specialtyName = useSpecialtyName(reference.specialties)
  const [table, setTable] = useState<PlatformTable>('gd')
  const [range, setRange] = useState({ from: months[Math.max(0, months.length - 12)], to: months[months.length - 1] })
  const [geo, setGeo] = useState<Geo>({})
  const [specialty, setSpecialty] = useState('')
  const [diagnosis, setDiagnosis] = useState<Diagnosis>({})

  const dims = dimensionsOf[table]
  const hasGeo = dims.includes('region')
  const hasSpecialty = dims.includes('specialty')
  const hasDiagnosis = dims.includes('group')
  const filter: Partial<Record<Dimension, string>> = {
    ...(hasGeo ? geoFilter(geo, reference.places) : {}),
    ...(hasSpecialty && specialty ? { specialty } : {}),
    ...(hasDiagnosis && diagnosis.group ? { group: diagnosis.group } : {}),
    ...(hasDiagnosis && diagnosis.subgroup ? { subgroup: diagnosis.subgroup } : {}),
  }
  const hasCities = (region?: string) => reference.places.some((p) => p.kind === 'city' && p.parent === region)
  const geoBy: Dimension | null = !hasGeo || geo.city ? null : filter.region ? (hasCities(filter.region) ? 'city' : null) : 'region'
  const diagnosisBy = hasDiagnosis ? drillOf(diagnosis, reference.groups) : null
  const specialtyBy = hasSpecialty && !specialty ? 'specialty' : null

  const key = JSON.stringify([table, range, filter])
  const cells = (by: Dimension | 'month' | null) => (by ? data.platformCells({ table, ...range, filter, by }) : Promise.resolve(null))
  const perMonth = useLoad(() => cells('month'), [data, key])
  const geoBars = useLoad(() => cells(geoBy), [data, key, geoBy])
  const diagnosisBars = useLoad(() => cells(diagnosisBy), [data, key, diagnosisBy])
  const specialtyBars = useLoad(() => cells(specialtyBy), [data, key, specialtyBy])

  const groupName = (code: string) => reference.groups.find((g) => g.code === code)?.name ?? code
  const unit = range.from === range.to ? t('analytics.patients') : t('analytics.patientMonths')
  // The heatmap takes a leaf region or an entity, the rises a leaf region; neither is narrowed on a table without
  // geography. A city asks for its region.
  const region = hasGeo ? (filter.region ?? reference.places.find((p) => p.id === geo.city)?.parent) : undefined
  const heatRegion = hasGeo ? (filter.entity ?? region) : undefined
  const periods = months.filter((m) => m >= range.from && m <= range.to) // the published months only

  function bars(state: typeof geoBars.state, title: string, toBar: (r: Row) => Bar, onSelect?: (key: string) => void) {
    return (
      <Loaded state={state}>
        {(c: PlatformCells | null) =>
          c && <BarChart title={title} bars={c.rows.map(toBar)} unit={unit} partial={c.partial} onSelect={onSelect} />
        }
      </Loaded>
    )
  }

  return (
    <div className="analytics-view">
      <p className="muted small">{t('analytics.platformNote')}</p>
      <div className="analytics-filters" role="group" aria-label={t('analytics.filters')}>
        <Choice label={t('analytics.view')} value={table} options={tables.map((v) => ({ value: v, label: t(`analytics.view.${v}`) }))} onChange={(v) => setTable(v as PlatformTable)} />
        <MonthRange months={months} from={range.from} to={range.to} onChange={setRange} />
        {hasGeo && <PlaceFilter places={reference.places} value={geo} onChange={setGeo} />}
        {hasSpecialty && <SpecialtyFilter specialties={reference.specialties} value={specialty} onChange={setSpecialty} />}
        {hasDiagnosis && <DiagnosisFilter groups={reference.groups} value={diagnosis} onChange={setDiagnosis} />}
      </div>
      <div className="analytics-grid">
        <Loaded state={perMonth.state}>
          {(c) => (
            <LineChart
              title={t('analytics.patientsPerMonth')}
              periods={periods}
              series={[{ key: 'patients', label: t('analytics.patients'), color: 'var(--series-1)', values: periods.map((p) => c?.rows.find((r) => r.key === p)?.count ?? null) }]}
            />
          )}
        </Loaded>
        {geoBy &&
          bars(geoBars.state, geoBy === 'city' ? t('analytics.byCity', { name: placeName(filter.region!) }) : t('analytics.byRegion'), named(placeName), (k) =>
            setGeo(geoBy === 'city' ? { place: geo.place, city: k } : { place: k }),
          )}
        {specialtyBy && bars(specialtyBars.state, t('analytics.bySpecialty'), named(specialtyName), setSpecialty)}
        {diagnosisBy && (
          <div className="analytics-wide">
            <DrillPath diagnosis={diagnosis} name={groupName} onChange={setDiagnosis} />
            {bars(
              diagnosisBars.state,
              diagnosisBy === 'group' ? t('analytics.topGroups') : diagnosisBy === 'subgroup' ? t('analytics.subgroupsOf', { name: groupName(diagnosis.group!) }) : t('analytics.codesIn', { name: groupName(diagnosis.subgroup ?? diagnosis.group!) }),
              (r) => diagnosisBar(r, diagnosisBy, groupName, t('analytics.noSubgroup')),
              diagnosisBy === 'code' ? undefined : (k) => setDiagnosis(drillInto(diagnosis, diagnosisBy, k)),
            )}
          </div>
        )}
        <PlatformHeatmap data={data} quarters={quarters} region={heatRegion} />
        <Rises data={data} region={region} groupName={groupName} placeName={placeName} />
      </div>
    </div>
  )
}

// PlatformHeatmap offers the published quarters (plan §5: built after the quarter), newest first.
function PlatformHeatmap({ data, quarters: published, region }: { data: AnalyticsData; quarters: string[]; region?: string }) {
  const t = useT()
  const quarters = [...published].reverse()
  const [quarter, setQuarter] = useState(quarters[0] ?? '')
  const cells = useLoad(() => (quarter ? data.platformHeatmap({ quarter, ...(region ? { region } : {}) }) : Promise.resolve({ cells: [] })), [data, quarter, region])
  if (quarters.length === 0) return null
  const quarterName = (q: string) => t('analytics.quarterName', { quarter: Number(q.slice(6)), year: Number(q.slice(0, 4)) })
  return (
    <div className="analytics-wide">
      <div className="analytics-filters">
        <Choice label={t('analytics.quarter')} value={quarter} options={quarters.map((q) => ({ value: q, label: quarterName(q) }))} onChange={setQuarter} />
      </div>
      <Loaded state={cells.state}>{(c) => <Heatmap title={t('analytics.platformHeatmap')} cells={c.cells} />}</Loaded>
    </div>
  )
}

type RisesProps = { data: AnalyticsData; region?: string; groupName: (code: string) => string; placeName: (id: string) => string }

// Rises lists the unusual rises of the last 8 weeks (plan §6): a flag with no count, with its disclaimer.
function Rises({ data, region, groupName, placeName }: RisesProps) {
  const t = useT()
  const rises = useLoad(() => data.platformRises({ ...(region ? { region } : {}), weeks: 8 }), [data, region])
  const title = t('analytics.rises')
  return (
    <section className="chart analytics-wide" aria-label={title}>
      <h2 className="chart-title">{title}</h2>
      <Loaded state={rises.state}>
        {(r) =>
          r.rises.length === 0 ? (
            <p className="muted">{t('analytics.noRises')}</p>
          ) : (
            <ul className="rises">
              {r.rises.map((rise) => (
                <li key={`${rise.week}-${rise.region}-${rise.group}`}>
                  <TipTag className="rise" label={t('analytics.rise')} tip={t('analytics.riseTip')} />
                  <span>
                    {t('analytics.riseLine', { group: groupName(rise.group), region: placeName(rise.region), week: Number(rise.week.slice(6)), year: Number(rise.week.slice(0, 4)) })}
                  </span>
                </li>
              ))}
            </ul>
          )
        }
      </Loaded>
    </section>
  )
}

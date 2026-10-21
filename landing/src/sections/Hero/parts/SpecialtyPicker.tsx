import { hero } from '../../../content/hero'
import type { Specialty } from '../../../content/specialties'
import { cx } from '../../../lib/cx'
import styles from './SpecialtyPicker.module.css'

type SpecialtyPickerProps = {
  specialties: readonly Specialty[]
  active: number
  onSelect: (index: number) => void
}

const twoDigits = (n: number) => String(n).padStart(2, '0')

// The specialty on show: its name, a line about it, and a dot per specialty to pick one.
export function SpecialtyPicker({ specialties, active, onSelect }: SpecialtyPickerProps) {
  return (
    <div className={styles.picker}>
      <div className={styles.count} style={{ '--delay': '1.9s' }}>
        <span>{twoDigits(active + 1)}</span> / {twoDigits(specialties.length)}
      </div>
      <div className={styles.names} style={{ '--delay': '1.9s' }} role="status" aria-live="polite">
        {specialties.map((s, i) => (
          <span key={s.name} className={cx(styles.name, i === active && styles.shown)} aria-hidden={i !== active || undefined}>
            {s.name}.
          </span>
        ))}
      </div>
      <div className={styles.blurbs} style={{ '--delay': '2s' }}>
        {specialties.map((s, i) => (
          <span key={s.name} className={cx(styles.blurb, i === active && styles.shown)} aria-hidden={i !== active || undefined}>
            {s.blurb}
          </span>
        ))}
      </div>
      <div className={styles.dots} style={{ '--delay': '2.1s' }}>
        {specialties.map((s, i) => (
          <button
            key={s.name}
            type="button"
            className={styles.dot}
            aria-label={`Show ${s.name.toLowerCase()}`}
            aria-pressed={i === active}
            onClick={() => onSelect(i)}
          >
            <span />
          </button>
        ))}
      </div>
      <p className={styles.note} style={{ '--delay': '2.2s' }}>
        {hero.note}
      </p>
    </div>
  )
}

import { hero } from '../../../content/hero'
import type { Specialty } from '../../../content/specialties'
import { cx } from '../../../lib/cx'
import { PhoneScreen } from './PhoneScreen'
import styles from './PhoneMockup.module.css'

type PhoneMockupProps = {
  specialties: readonly Specialty[]
  active: number
  swap: 'a' | 'b'
}

export function PhoneMockup({ specialties, active, swap }: PhoneMockupProps) {
  return (
    <div className={styles.float}>
      <div className={cx(styles.phone, swap === 'a' ? styles.swapA : styles.swapB)} role="img" aria-label={hero.phone.label}>
        <div className={styles.screens}>
          {specialties.map((specialty, i) => (
            <PhoneScreen key={specialty.name} specialty={specialty} active={i === active} />
          ))}
        </div>
      </div>
    </div>
  )
}

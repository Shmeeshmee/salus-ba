import { hero } from '../../../content/hero'
import { cx } from '../../../lib/cx'
import styles from './FloatingChips.module.css'

const { booking, form } = hero.chips

// Two notification cards that float beside the phone. Decoration only.
export function FloatingChips() {
  return (
    <>
      <div className={cx(styles.chip, styles.booking)} style={{ '--delay': '2.3s' }} aria-hidden="true">
        <div className={styles.bookingCard} style={{ '--float-delay': '-2s' }}>
          <span className={styles.check}>
            <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
              <path d="M3 8.5l3 3 7-7" />
            </svg>
          </span>
          <span className={styles.text}>
            {booking.title}
            <br />
            <span className={styles.detail}>{booking.detail}</span>
          </span>
        </div>
      </div>
      <div className={cx(styles.chip, styles.form)} style={{ '--delay': '2.5s' }} aria-hidden="true">
        <div className={styles.formCard} style={{ '--float-delay': '-4s' }}>
          <div className={styles.formHead}>
            <span>{form.title}</span>
            <span className={styles.detail}>{form.progress}</span>
          </div>
          <div className={styles.bar}>
            <div className={styles.fill} style={{ width: `${form.percent}%` }} />
          </div>
        </div>
      </div>
    </>
  )
}

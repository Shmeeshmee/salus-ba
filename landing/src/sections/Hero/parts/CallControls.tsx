import { cx } from '../../../lib/cx'
import styles from './CallControls.module.css'

export function CallControls() {
  return (
    <div className={styles.controls}>
      <span className={styles.control}>
        <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
          <rect x="6.5" y="2" width="5" height="9" rx="2.5" />
          <path d="M3.5 8.5a5.5 5.5 0 0 0 11 0M9 14v2.5" />
        </svg>
      </span>
      <span className={styles.control}>
        <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round">
          <rect x="1.5" y="4.5" width="10.5" height="9" rx="2" />
          <path d="M12 8l4.5-2.5v7L12 10z" />
        </svg>
      </span>
      <span className={cx(styles.control, styles.end)}>
        <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round">
          <path d="M4 4l10 10M14 4L4 14" />
        </svg>
      </span>
    </div>
  )
}

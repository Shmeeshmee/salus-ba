import { cx } from '../../../lib/cx'
import { ecgPath } from '../../../lib/ecgPath'
import { MonitorReadouts } from './MonitorReadouts'
import styles from './EcgMonitor.module.css'

const TRACE = ecgPath()

// Background of the hero: a patient monitor whose trace is redrawn by a full-height sweep.
export function EcgMonitor() {
  return (
    <div className={styles.monitor} aria-hidden="true">
      <div className={styles.grid} />
      <div className={styles.band}>
        <svg className={cx(styles.trace, styles.old)} viewBox="0 0 2400 240" preserveAspectRatio="none">
          <path className={styles.path} d={TRACE} strokeWidth="2" />
        </svg>
        <svg className={cx(styles.trace, styles.fresh)} viewBox="0 0 2400 240" preserveAspectRatio="none">
          <path className={styles.path} d={TRACE} strokeWidth="2.2" />
        </svg>
      </div>
      <div className={styles.sweep} />
      <MonitorReadouts />
    </div>
  )
}

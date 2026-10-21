import { hero } from '../../../content/hero'
import { cx } from '../../../lib/cx'
import styles from './MonitorReadouts.module.css'

// The vital signs and lead settings printed on the monitor. Hidden on small screens.
export function MonitorReadouts() {
  return (
    <>
      <div className={styles.readouts}>
        {hero.monitor.readouts.map((readout) => (
          <div key={readout.label}>
            <div className={styles.label}>{readout.label}</div>
            <div className={cx(styles.value, readout.beat && styles.pulsing)}>
              {readout.value}
              {readout.beat && <span className={styles.beat} />}
              {readout.unit && <span className={styles.unit}>{readout.unit}</span>}
            </div>
          </div>
        ))}
      </div>
      <div className={styles.lead}>{hero.monitor.lead}</div>
    </>
  )
}

import { hero } from '../../../content/hero'
import { cx } from '../../../lib/cx'
import styles from './VideoTile.module.css'

type VideoTileProps = {
  doctor: string
  initials: string
}

// The doctor's video: their initials with two rings pinging out, and the patient's own picture in a corner.
export function VideoTile({ doctor, initials }: VideoTileProps) {
  return (
    <div className={styles.video}>
      <span className={styles.ping} />
      <span className={cx(styles.ping, styles.pingLater)} />
      <span className={styles.avatar}>{initials}</span>
      <span className={styles.self}>{hero.phone.self}</span>
      <span className={styles.doctor}>{doctor}</span>
    </div>
  )
}

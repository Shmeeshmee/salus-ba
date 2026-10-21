import { hero } from '../../../content/hero'
import type { Specialty } from '../../../content/specialties'
import { cx } from '../../../lib/cx'
import { CallControls } from './CallControls'
import { VideoTile } from './VideoTile'
import styles from './PhoneScreen.module.css'

type PhoneScreenProps = {
  specialty: Specialty
  active: boolean
}

const WAVE_DELAYS_S = [0, 0.15, 0.3, 0.45, 0.6, 0.25]

// One video consult screen. All four are stacked in the phone; the active one fades in.
export function PhoneScreen({ specialty, active }: PhoneScreenProps) {
  const { palette } = specialty
  return (
    <div
      className={cx(styles.screen, active && styles.active)}
      style={{ '--screen-accent': palette.accent, '--screen-tint': palette.tint, '--screen-ink': palette.ink }}
      aria-hidden="true"
    >
      <div className={styles.top}>
        <span className={styles.brand}>{hero.phone.brand}</span>
        <span className={styles.live}>
          <span className={styles.liveDot} />
          {hero.phone.live}
        </span>
      </div>
      <VideoTile doctor={specialty.doctor} initials={specialty.initials} />
      <div className={styles.transcription}>
        <span className={styles.transcriptionLabel}>{hero.phone.transcription}</span>
        <span className={styles.wave}>
          {WAVE_DELAYS_S.map((delay) => (
            <i key={delay} style={{ '--delay': `${delay}s` }} />
          ))}
        </span>
      </div>
      <p className={styles.quote}>{specialty.quote}</p>
      <CallControls />
    </div>
  )
}

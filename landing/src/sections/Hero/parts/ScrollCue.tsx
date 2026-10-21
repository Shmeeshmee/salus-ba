import { hero } from '../../../content/hero'
import styles from './ScrollCue.module.css'

type ScrollCueProps = {
  onClick: () => void
}

export function ScrollCue({ onClick }: ScrollCueProps) {
  return (
    <button
      type="button"
      className={styles.cue}
      style={{ '--delay': '2.6s' }}
      aria-label={hero.scrollCue.ariaLabel}
      onClick={onClick}
    >
      <span>{hero.scrollCue.label}</span>
      <span className={styles.track}>
        <span className={styles.line} />
      </span>
    </button>
  )
}

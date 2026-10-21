import { Fragment } from 'react'
import { highlights } from '../../content/highlights'
import styles from './Highlights.module.css'

// The marquee needs the list twice to loop seamlessly; the second copy is hidden from assistive technology.
const COPIES = [false, true]

export function Highlights() {
  return (
    <section className={styles.highlights} aria-label={highlights.label}>
      <div className={styles.track}>
        {COPIES.map((hidden) =>
          highlights.items.map((item) => (
            <Fragment key={`${hidden}-${item}`}>
              <span aria-hidden={hidden || undefined}>{item}</span>
              <span className={styles.separator} aria-hidden="true">
                ●
              </span>
            </Fragment>
          )),
        )}
      </div>
    </section>
  )
}

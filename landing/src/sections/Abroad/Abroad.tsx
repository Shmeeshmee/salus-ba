import { Container } from '../../components/Container'
import { SectionHeading } from '../../components/SectionHeading'
import { abroad } from '../../content/abroad'
import styles from './Abroad.module.css'

export function Abroad() {
  return (
    <Container id={abroad.id} className={styles.abroad}>
      <SectionHeading eyebrow={abroad.eyebrow} title={abroad.heading} />
      <div className={styles.panel}>
        <p className={styles.body}>{abroad.body}</p>
        <ul className={styles.points}>
          {abroad.points.map((point) => (
            <li key={point.text} className={styles.point}>
              <span>{point.text}</span>
              {point.coming && <span className={styles.coming}>{abroad.comingLabel}</span>}
            </li>
          ))}
        </ul>
      </div>
    </Container>
  )
}

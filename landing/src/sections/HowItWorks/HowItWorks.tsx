import { Container } from '../../components/Container'
import { SectionHeading } from '../../components/SectionHeading'
import { howItWorks } from '../../content/steps'
import styles from './HowItWorks.module.css'

export function HowItWorks() {
  return (
    <Container id={howItWorks.id} className={styles.how}>
      <SectionHeading eyebrow={howItWorks.eyebrow} title={howItWorks.heading} />
      <ol className={styles.steps}>
        {howItWorks.steps.map((step, i) => (
          <li key={step.title} className={styles.step}>
            <span className={styles.number}>{String(i + 1).padStart(2, '0')}</span>
            <h3 className={styles.title}>{step.title}</h3>
            <p className={styles.body}>{step.body}</p>
          </li>
        ))}
      </ol>
    </Container>
  )
}

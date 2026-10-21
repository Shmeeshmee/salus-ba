import { Container } from '../../components/Container'
import { SectionHeading } from '../../components/SectionHeading'
import { pricing } from '../../content/pricing'
import styles from './Pricing.module.css'

export function Pricing() {
  return (
    <Container id={pricing.id} className={styles.pricing}>
      <SectionHeading eyebrow={pricing.eyebrow} title={pricing.heading} />
      <div className={styles.plans}>
        {pricing.plans.map((plan) => (
          <div key={plan.label} className={styles.plan}>
            <div className={styles.label}>{plan.label}</div>
            <div className={styles.amount}>
              {plan.amount}
              {plan.unit && <span className={styles.unit}>{plan.unit}</span>}
            </div>
            <p className={styles.body}>{plan.body}</p>
          </div>
        ))}
      </div>
    </Container>
  )
}

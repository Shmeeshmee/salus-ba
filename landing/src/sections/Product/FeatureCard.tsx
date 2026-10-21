import type { Feature } from '../../content/features'
import { cx } from '../../lib/cx'
import { FeatureIcon } from './FeatureIcon'
import styles from './FeatureCard.module.css'

type FeatureCardProps = {
  feature: Feature
}

export function FeatureCard({ feature }: FeatureCardProps) {
  return (
    <article className={cx(styles.card, feature.dark && styles.dark)}>
      <FeatureIcon name={feature.icon} className={styles.icon} />
      <h3 className={styles.title}>{feature.title}</h3>
      <p className={styles.body}>{feature.body}</p>
    </article>
  )
}

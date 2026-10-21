import { Container } from '../../components/Container'
import { SectionHeading } from '../../components/SectionHeading'
import { product } from '../../content/features'
import { FeatureCard } from './FeatureCard'
import styles from './Product.module.css'

export function Product() {
  return (
    <Container id={product.id} className={styles.product}>
      <SectionHeading eyebrow={product.eyebrow} title={product.heading} />
      <div className={styles.grid}>
        {product.features.map((feature) => (
          <FeatureCard key={feature.title} feature={feature} />
        ))}
      </div>
    </Container>
  )
}

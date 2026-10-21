import { ArrowIcon } from '../../components/ArrowIcon'
import { Button } from '../../components/Button'
import { Container } from '../../components/Container'
import { demo } from '../../content/demo'
import styles from './Demo.module.css'

export function Demo() {
  return (
    <Container id={demo.id} className={styles.demo}>
      <div className={styles.panel}>
        <div className={styles.orbitFrame} aria-hidden="true">
          <div className={styles.orbit}>
            <span className={styles.satellite} />
          </div>
        </div>
        <h2 className={styles.title}>{demo.heading}</h2>
        <Button variant="mint" href={demo.cta.href} className={styles.cta}>
          {demo.cta.label}
          <ArrowIcon strokeWidth={1.8} />
        </Button>
      </div>
    </Container>
  )
}

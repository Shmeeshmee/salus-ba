import { Container } from '../../components/Container'
import { Eyebrow } from '../../components/Eyebrow'
import { standards } from '../../content/standards'
import styles from './Standards.module.css'

export function Standards() {
  return (
    <Container id={standards.id} className={styles.standards}>
      <div className={styles.panel}>
        <div className={styles.text}>
          <Eyebrow>{standards.eyebrow}</Eyebrow>
          <h2 className={styles.title}>{standards.heading}</h2>
          <p className={styles.body}>{standards.body}</p>
        </div>
        <div className={styles.tiles}>
          {standards.tiles.map((tile) => (
            <div key={tile.title} className={styles.tile}>
              <b className={styles.tileTitle}>{tile.title}</b>
              <span className={styles.tileDetail}>{tile.detail}</span>
            </div>
          ))}
        </div>
      </div>
    </Container>
  )
}

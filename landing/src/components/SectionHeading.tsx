import { Eyebrow } from './Eyebrow'
import styles from './SectionHeading.module.css'

type SectionHeadingProps = {
  eyebrow: string
  title: string
}

export function SectionHeading({ eyebrow, title }: SectionHeadingProps) {
  return (
    <div className={styles.heading}>
      <Eyebrow>{eyebrow}</Eyebrow>
      <h2 className={styles.title}>{title}</h2>
    </div>
  )
}

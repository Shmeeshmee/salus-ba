import styles from './SkipLink.module.css'

type SkipLinkProps = {
  targetId: string
  children: string
}

export function SkipLink({ targetId, children }: SkipLinkProps) {
  return (
    <a className={styles.skip} href={`#${targetId}`}>
      {children}
    </a>
  )
}

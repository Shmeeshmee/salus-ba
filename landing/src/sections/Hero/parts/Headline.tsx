import { Fragment } from 'react'
import { ArrowIcon } from '../../../components/ArrowIcon'
import { Button } from '../../../components/Button'
import { hero } from '../../../content/hero'
import styles from './Headline.module.css'

type HeadlineProps = {
  onSeeProduct: () => void
}

const FIRST_WORD_DELAY_S = 1.55
const WORD_STAGGER_S = 0.08

export function Headline({ onSeeProduct }: HeadlineProps) {
  let word = 0
  return (
    <div className={styles.headline}>
      <p className={styles.eyebrow} style={{ '--delay': '1.6s' }}>
        {hero.eyebrow}
      </p>
      <h1 className={styles.title}>
        {hero.headline.map((line, l) => (
          <Fragment key={line.join(' ')}>
            {l > 0 && (
              <>
                {' '}
                <br />
              </>
            )}
            {line.map((text, w) => (
              <Fragment key={text}>
                {w > 0 && ' '}
                <span className={styles.mask}>
                  <span
                    className={styles.word}
                    style={{ '--delay': `${(FIRST_WORD_DELAY_S + WORD_STAGGER_S * word++).toFixed(2)}s` }}
                  >
                    {text}
                  </span>
                </span>
              </Fragment>
            ))}
          </Fragment>
        ))}
      </h1>
      <div className={styles.actions} style={{ '--delay': '2.1s' }}>
        <Button variant="ink" href={hero.primaryCta.href}>
          {hero.primaryCta.label}
          <ArrowIcon strokeWidth={1.6} />
        </Button>
        <Button variant="line" onClick={onSeeProduct}>
          {hero.secondaryCta}
        </Button>
      </div>
    </div>
  )
}

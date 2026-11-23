import type { Condition, Item } from './builder'
import { useList, useT } from './i18n/useT'

type Props = {
  item: Item // a yes or no or a choice question
  label: (it: Item) => string
  followUps: (answer: Condition['answer']) => Item[]
  onFollowUp: (answer: Condition['answer']) => void
}

// BuilderBranch is "If the answer is X, ask ..." on the question answered (SAL-43): per answer, what it asks
// and a button that adds a question shown only on that answer, at the end of the question's branches.
export function BuilderBranch({ item, label, followUps, onFollowUp }: Props) {
  const t = useT()
  const list = useList()
  const answers: { value: Condition['answer']; text: string }[] =
    item.kind === 'boolean'
      ? [
          { value: true, text: t('questionnaire.yes') },
          { value: false, text: t('questionnaire.no') },
        ]
      : item.options.filter((o) => o.trim() !== '').map((o) => ({ value: o, text: o }))
  if (answers.length === 0) return null
  return (
    <fieldset className="branches">
      <legend>{t('questionnaire.branches')}</legend>
      <ul>
        {answers.map((a) => {
          const asks = followUps(a.value)
          return (
            <li key={String(a.value)} className="branch">
              <span className="branch-if">{t('questionnaire.branchIf', { answer: a.text })}</span>
              {asks.length > 0 && <span className="muted"> {list(asks.map(label))}</span>}
              <button type="button" className="quiet" aria-label={t('questionnaire.branchAskLabel', { answer: a.text })} onClick={() => onFollowUp(a.value)}>
                {t('questionnaire.branchAsk')}
              </button>
            </li>
          )
        })}
      </ul>
    </fieldset>
  )
}

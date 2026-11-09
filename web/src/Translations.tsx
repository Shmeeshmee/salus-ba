import { useId, useState } from 'react'
import { all, isChoice, maxDescription, maxHelp, maxOption, maxText, maxTitle, textLabel, type Item } from './builder'
import { useT } from './i18n/useT'
import { contentLanguageNames as names, type ContentLanguage, type Translation } from './questionnaire'
import { progress, unused, withItem } from './translation'
import { Confirm } from './ui'
import './translation.css'

type T = ReturnType<typeof useT>

type LanguagesProps = {
  language: ContentLanguage // the original
  translations: Translation[]
  description: string
  items: Item[]
  editing?: ContentLanguage // the translation open beside the original; none while the items are edited
  adding: boolean
  onLanguage: (l: ContentLanguage) => void
  onEdit: (l?: ContentLanguage) => void
  onAdding: (on: boolean) => void
  onAdd: (l: ContentLanguage) => void
  onRemove: (l: ContentLanguage) => void
}

// statusOf is B4: how many of the texts a translation has, or that it is complete.
function statusOf(t: Translation, description: string, items: Item[], tr: T): string {
  const p = progress(t, description, items)
  const language = names[t.language]
  return p.n === p.total ? tr('questionnaire.translationComplete', { language }) : tr('questionnaire.translationProgress', { language, n: p.n, total: p.total })
}

// Languages is the builder's languages (B1 to B4): the original language, and per translation how far it is,
// each a button that opens it beside the original.
export function Languages(p: LanguagesProps) {
  const t = useT()
  const id = useId()
  const [removing, setRemoving] = useState<ContentLanguage>()
  const offered = unused(p.language, p.translations)
  const originals = [p.language, ...offered]
  return (
    <section className="languages" aria-labelledby={`${id}-h`}>
      <h2 id={`${id}-h`}>{t('questionnaire.languages')}</h2>
      <div className="field">
        <label htmlFor={`${id}-original`}>{t('questionnaire.originalLanguage')}</label>
        <select id={`${id}-original`} value={p.language} onChange={(e) => p.onLanguage(e.target.value as ContentLanguage)}>
          {originals.map((l) => (
            <option key={l} value={l} lang={l}>
              {names[l]}
            </option>
          ))}
        </select>
      </div>
      {p.translations.length > 0 && (
        <ul className="language-views">
          <li>
            <button type="button" className="quiet" aria-pressed={p.editing === undefined} onClick={() => p.onEdit(undefined)}>
              {t('questionnaire.originalColumn', { language: names[p.language] })}
            </button>
          </li>
          {p.translations.map((tr) => (
            <li key={tr.language}>
              <button type="button" id={`${id}-${tr.language}`} className="quiet" aria-pressed={p.editing === tr.language} onClick={() => p.onEdit(tr.language)}>
                {statusOf(tr, p.description, p.items, t)}
              </button>
              <button type="button" className="quiet danger-text" aria-describedby={`${id}-${tr.language}`} onClick={() => setRemoving(tr.language)}>
                {t('questionnaire.removeTranslation')}
              </button>
            </li>
          ))}
        </ul>
      )}
      {removing && (
        <Confirm
          question={t('questionnaire.removeTranslationQuestion', { language: names[removing] })}
          lines={[t('questionnaire.removeTranslationWarning')]}
          confirm={t('questionnaire.removeConfirm')}
          cancel={t('questionnaire.cancel')}
          danger
          onConfirm={() => {
            p.onRemove(removing)
            setRemoving(undefined)
          }}
          onCancel={() => setRemoving(undefined)}
        />
      )}
      {offered.length > 0 && !p.adding && (
        <div className="actions">
          <button type="button" onClick={() => p.onAdding(true)}>
            {t('questionnaire.addTranslation')}
          </button>
        </div>
      )}
      {p.adding && (
        <fieldset className="add-translation">
          <legend>{t('questionnaire.addTranslation')}</legend>
          <div className="actions">
            {offered.map((l, i) => (
              <button type="button" key={l} lang={l} autoFocus={i === 0} onClick={() => p.onAdd(l)}>
                {names[l]}
              </button>
            ))}
            <button type="button" className="quiet" onClick={() => p.onAdding(false)}>
              {t('questionnaire.cancel')}
            </button>
          </div>
        </fieldset>
      )}
    </section>
  )
}

type EditorProps = {
  language: ContentLanguage
  translation: Translation
  title: string
  description: string
  items: Item[]
  onChange: (t: Translation) => void
}

// TranslationEditor is one translation beside the original (B8): per text, the original on the left and its
// translation on the right; stacked on a narrow screen.
export function TranslationEditor({ language, translation: tr, title, description, items, onChange }: EditorProps) {
  const t = useT()
  const id = useId()
  const row = { head: `${id}-tr`, lang: tr.language }
  const sections = new Set(items.filter((it) => it.kind === 'group').flatMap((s) => s.items.map((c) => c.linkId)))
  return (
    <div className="translation">
      <div className="translation-head">
        <p>{t('questionnaire.originalColumn', { language: names[language] })}</p>
        <p id={row.head} lang={tr.language}>
          {names[tr.language]}
        </p>
      </div>
      <ul className="translation-rows">
        <Row {...row} label={t('questionnaire.titleLabel')} original={title} value={tr.title} max={maxTitle} onChange={(x) => onChange({ ...tr, title: x })} />
        {description.trim() !== '' && (
          <Row {...row} label={t('questionnaire.descriptionLabel')} original={description} value={tr.description ?? ''} max={maxDescription} long onChange={(x) => onChange({ ...tr, description: x })} />
        )}
        {all(items).map((it) => {
          const e = tr.items[it.linkId] ?? {}
          const set = (patch: Parameters<typeof withItem>[2]) => onChange(withItem(tr, it.linkId, patch))
          const option = (i: number, x: string) => set({ options: it.options.map((_, j) => (j === i ? x : (e.options?.[j] ?? ''))) })
          return (
            <li key={it.linkId} className={sections.has(it.linkId) ? 'translation-item nested' : 'translation-item'}>
              <ul>
                <Row {...row} label={t(textLabel(it.kind))} original={it.text} value={e.text ?? ''} max={maxText} long={it.kind === 'display'} onChange={(x) => set({ text: x })} />
                {it.kind !== 'display' && it.help.trim() !== '' && (
                  <Row {...row} label={t('questionnaire.help')} original={it.help} value={e.help ?? ''} max={maxHelp} long onChange={(x) => set({ help: x })} />
                )}
                {isChoice(it.kind) &&
                  it.options.map((o, i) => (
                    // Options are translated by position, as they are saved.
                    <Row key={i} {...row} label={t('questionnaire.option', { n: i + 1 })} original={o} value={e.options?.[i] ?? ''} max={maxOption} onChange={(x) => option(i, x)} />
                  ))}
              </ul>
            </li>
          )
        })}
      </ul>
    </div>
  )
}

type RowProps = { head: string; lang: string; label: string; original: string; value: string; max: number; long?: boolean; onChange: (v: string) => void }

// Row is one text: its kind and the original, then the translation's field, named by the kind and the
// translation's language and described by the original.
function Row({ head, lang, label, original, value, max, long, onChange }: RowProps) {
  const t = useT()
  const id = useId()
  const near = value.length >= max * 0.8
  const field = {
    value,
    lang,
    dir: 'auto',
    maxLength: max,
    'aria-labelledby': `${id}-label ${head}`,
    'aria-describedby': near ? `${id}-original ${id}-count` : `${id}-original`,
  }
  return (
    <li className="translation-row">
      <div className="original">
        <span className="hint" id={`${id}-label`}>
          {label}
        </span>
        <p className="original-text" id={`${id}-original`}>
          {original}
        </p>
      </div>
      <div className="field">
        {long ? <textarea {...field} rows={3} onChange={(e) => onChange(e.target.value)} /> : <input {...field} onChange={(e) => onChange(e.target.value)} />}
        {near && (
          <p className="hint" id={`${id}-count`}>
            {t('questionnaire.characters', { n: value.length, max })}
          </p>
        )}
      </div>
    </li>
  )
}

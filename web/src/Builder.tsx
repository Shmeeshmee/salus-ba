import { useEffect, useRef, useState, type ReactNode } from 'react'
import type { ApiClient, ErrorCode } from './api/client'
import { FhirError, type Bundle, type FhirClient } from './api/fhir'
import { attempt, type Outcome } from './api/result'
import {
  at,
  canStep,
  changeKind,
  count,
  dependents,
  earlierQuestions,
  find,
  insert,
  kinds,
  maxDescription,
  maxItems,
  maxTitle,
  newItem,
  problems,
  remove,
  rowsOf,
  step,
  update,
  type Item,
  type Kind,
} from './builder'
import { DragArea, ListEnd, PaletteSource, SortableList, SortableRow } from './builder-dnd'
import { ChevronIcon } from './icons'
import { FilesSummary, ItemCard, TextField } from './ItemCard'
import { useLoad } from './load'
import { useT } from './i18n/useT'
import { contentLanguageNames, fromFHIR, fromJSON, toRequest, type ContentLanguage, type FHIRQuestionnaire, type Translation, type Version } from './questionnaire'
import { guardLeaving, navigate } from './route'
import { firstIncomplete, progress } from './translation'
import { Languages, TranslationEditor } from './Translations'
import { Alert, Confirm, Link } from './ui'

type Loaded = { version: Version; series: Version[] }
type Props = { api: ApiClient; fhir: FhirClient; id: string }

type T = ReturnType<typeof useT>

// labelOf names an item by its text, or by its type while it has none.
const labelOf = (t: T) => (it: Item) => it.text.trim() || t(`questionnaire.kinds.${it.kind}`)
const newLinkId = () => crypto.randomUUID()

// Builder is the page of one questionnaire version: a draft is edited (the palette, the items, save,
// publish, discard); an active version offers "Edit as new version" and "Retire"; a retired one is read only.
export function Builder({ api, fhir, id }: Props) {
  const t = useT()
  const { state, reload, set } = useLoad(async (): Promise<Loaded> => {
    const q = await fhir.get<FHIRQuestionnaire>(`Questionnaire/${encodeURIComponent(id)}`)
    const bundle = await fhir.get<Bundle<FHIRQuestionnaire>>(`Questionnaire?${new URLSearchParams({ url: q.url })}`)
    return { version: fromFHIR(q), series: (bundle.entry ?? []).map((e) => fromFHIR(e.resource)) }
  }, [fhir, id])

  if (state.name === 'loading') return <Page>{<p className="muted">{t('common.loading')}</p>}</Page>
  if (state.name === 'failed') {
    const gone = state.error instanceof FhirError && (state.error.status === 404 || state.error.status === 403)
    return <Page>{<Alert message={t(gone ? 'questionnaire.notAvailable' : 'common.somethingWrong')} />}</Page>
  }
  const loaded = state.value
  const changed = (version: Version) => set({ ...loaded, version })
  if (loaded.version.status === 'draft') return <Editor api={api} loaded={loaded} onReload={reload} onPublished={changed} />
  return <Published api={api} loaded={loaded} onChanged={changed} />
}

function Page({ children, wide }: { children: ReactNode; wide?: boolean }) {
  const t = useT()
  return (
    <section className={wide ? 'panel builder-page' : 'panel'}>
      <Link to="/questionnaires" className="back">
        <ChevronIcon direction="left" />
        {t('questionnaire.title')}
      </Link>
      {children}
    </section>
  )
}

type Failure = { message: string; values?: Record<string, string>; reload?: boolean; draft?: string } // message: a catalog message id

// failure maps an API refusal to its W29 text's message id.
function failure(code: ErrorCode | undefined, series: Version[]): Failure {
  switch (code) {
    case 'version_conflict':
      return { message: 'questionnaire.changed', reload: true }
    case 'not_draft':
      return { message: 'questionnaire.notDraft' }
    case 'invalid_questionnaire':
      return { message: 'questionnaire.needsAttention' }
    case 'not_found':
      return { message: 'questionnaire.notAvailable' }
    case 'draft_exists':
      return { message: 'questionnaire.draftExists', draft: series.find((v) => v.status === 'draft')?.id }
    default:
      return { message: 'common.somethingWrong' }
  }
}

function FailureAlert({ failure, onReload }: { failure?: Failure; onReload?: () => void }) {
  const t = useT()
  if (!failure) return null
  return (
    <div className="failure">
      <Alert message={t(failure.message, failure.values)} />
      {failure.reload && onReload && (
        <div className="actions">
          <button type="button" onClick={onReload}>
            {t('questionnaire.reload')}
          </button>
        </div>
      )}
      {failure.draft && (
        <div className="actions">
          <button type="button" onClick={() => navigate(`/questionnaires/${encodeURIComponent(failure.draft ?? '')}`)}>
            {t('questionnaire.openDraft')}
          </button>
        </div>
      )}
    </div>
  )
}

type EditorProps = { api: ApiClient; loaded: Loaded; onReload: () => void; onPublished: (v: Version) => void }

function Editor({ api, loaded, onReload, onPublished }: EditorProps) {
  const t = useT()
  const label = labelOf(t)
  const [title, setTitle] = useState(loaded.version.title)
  const [description, setDescription] = useState(loaded.version.description)
  const [items, setItems] = useState(loaded.version.items)
  const [language, setLanguage] = useState(loaded.version.language)
  const [translations, setTranslations] = useState(loaded.version.translations)
  const [editing, setEditing] = useState<ContentLanguage>() // the translation open beside the original
  const [adding, setAdding] = useState(false)
  const [incomplete, setIncomplete] = useState<ContentLanguage>() // the translation a publish was refused for
  const [versionId, setVersionId] = useState(loaded.version.versionId)
  const [dirty, setDirty] = useState(false)
  const [saved, setSaved] = useState(false)
  const [busy, setBusy] = useState(false)
  const [publishing, setPublishing] = useState(false)
  const [opened, setOpened] = useState<string[]>([])
  const [flagged, setFlagged] = useState<string[]>([])
  const [failed, setFailed] = useState<Failure>()
  const [confirm, setConfirm] = useState<'translate' | 'publish' | 'discard'>()
  const [leavingTo, setLeavingTo] = useState<string>()
  const [announcement, setAnnouncement] = useState('')
  const [fresh, setFresh] = useState<string>() // the item just added, whose text takes the focus
  const dirtyRef = useRef(false)
  const edits = useRef(0) // every change counts: a save clears dirty only if none came while it was in flight
  const v = loaded.version

  useEffect(() => {
    dirtyRef.current = dirty
  }, [dirty])

  useEffect(() => {
    const release = guardLeaving((to) => {
      if (!dirtyRef.current) return true
      setLeavingTo(to)
      return false
    })
    const onUnload = (e: BeforeUnloadEvent) => {
      if (dirtyRef.current) e.preventDefault()
    }
    globalThis.addEventListener('beforeunload', onUnload)
    return () => {
      release()
      globalThis.removeEventListener('beforeunload', onUnload)
    }
  }, [])

  function changed() {
    edits.current++
    setDirty(true)
    dirtyRef.current = true
    setSaved(false)
  }

  function edit(next: Item[]) {
    setItems(next)
    changed()
  }

  function leave(to: string) {
    dirtyRef.current = false
    navigate(to)
  }

  // save answers the saved version, or nothing when the save was refused.
  async function save(): Promise<Version | undefined> {
    setBusy(true)
    setFailed(undefined)
    setSaved(false)
    const sent = edits.current
    const res = await attempt(
      api.PUT('/questionnaires/{id}', {
        params: { path: { id: v.id }, header: { 'If-Match': `W/"${versionId}"` } },
        body: toRequest(title, description, items, language, translations),
      }),
    )
    setBusy(false)
    if (!res.ok) {
      refused(res)
      return undefined
    }
    const saved = fromJSON(res.data)
    setVersionId(saved.versionId)
    setFlagged([])
    if (edits.current === sent) {
      // The answer's translations lack the texts whose original changed (docs/i18n/PROGRESS.md 1.4).
      setTranslations(saved.translations)
      setDirty(false)
      dirtyRef.current = false
      setSaved(true)
    }
    return saved
  }

  // refused shows a refusal; now is the translations as the API holds them (after a save, its answer's).
  function refused(res: Outcome<unknown>, now = translations) {
    if (res.ok) return
    setFailed(failure(res.code, loaded.series))
    if (res.code === 'translation_incomplete') refusedTranslation(res.linkIds ?? [], now)
    if (res.code !== 'invalid_questionnaire') return
    const ids = res.linkIds ?? []
    setFlagged(ids)
    setOpened((o) => [...new Set([...o, ...ids])])
  }

  // refusedTranslation shows B6 and the items the API named. The error's language is not passed on by
  // attempt; it is the translation the same rule finds incomplete here, which the API checks first too.
  function refusedTranslation(linkIds: string[], now: Translation[]) {
    setFlagged(linkIds)
    const first = firstIncomplete(now, description, items)
    if (first) showIncomplete(first.language)
    else setFailed({ message: 'questionnaire.needsAttention' })
  }

  // showIncomplete is B6, and B7 on the items the translation lacks a text of.
  function showIncomplete(l: ContentLanguage) {
    setIncomplete(l)
    setFailed({ message: 'questionnaire.translationIncomplete', values: { language: contentLanguageNames[l] } })
  }

  // askPublish checks the draft as the API will before it asks: a draft with problems or an incomplete
  // translation is shown, not offered; one with no translation is asked about first (B5).
  function askPublish() {
    const ids = [...found.keys()]
    if (ids.length > 0) {
      setFailed({ message: 'questionnaire.needsAttention' })
      setOpened((o) => [...new Set([...o, ...ids])])
      return
    }
    const first = firstIncomplete(translations, description, items)
    if (first) return showIncomplete(first.language)
    setFailed(undefined)
    setConfirm(translations.length === 0 ? 'translate' : 'publish')
  }

  function editTranslations(next: Translation[]) {
    setTranslations(next)
    changed()
  }

  function addTranslation(l: ContentLanguage) {
    editTranslations([...translations, { language: l, title: '', items: {} }])
    setAdding(false)
    setEditing(l)
  }

  function removeTranslation(l: ContentLanguage) {
    editTranslations(translations.filter((x) => x.language !== l))
    if (editing === l) setEditing(undefined)
    if (incomplete === l) {
      setIncomplete(undefined)
      setFailed(undefined)
    }
  }

  // publish saves what is unsaved, then publishes that version. The items are out of reach meanwhile (inert), so
  // no edit can come after the save and be lost to the published version.
  async function publish() {
    setPublishing(true)
    const current = dirty ? await save() : { versionId, translations }
    if (current === undefined) {
      setPublishing(false)
      setConfirm(undefined)
      return
    }
    setBusy(true)
    const res = await attempt(
      api.POST('/questionnaires/{id}/publish', { params: { path: { id: v.id }, header: { 'If-Match': `W/"${current.versionId}"` } } }),
    )
    setBusy(false)
    setPublishing(false)
    setConfirm(undefined)
    if (res.ok) onPublished(fromJSON(res.data))
    else refused(res, current.translations)
  }

  async function discard() {
    setBusy(true)
    setFailed(undefined)
    const res = await attempt(api.DELETE('/questionnaires/{id}', { params: { path: { id: v.id } } }))
    setBusy(false)
    setConfirm(undefined)
    if (res.ok) leave('/questionnaires')
    else refused(res)
  }

  function add(kind: Kind) {
    const it = newItem(kind, newLinkId())
    const res = insert(items, it)
    if (!res.ok) return
    edit(res.items)
    setOpened((o) => [...o, it.linkId])
    setFresh(it.linkId)
  }

  function placed(next: Item[], linkId: string) {
    edit(next)
    if (find(items, linkId)) return
    setOpened((o) => [...o, linkId])
    setFresh(linkId)
  }

  function stepItem(linkId: string, dir: -1 | 1) {
    const res = step(items, linkId, dir)
    if (!res?.ok) return
    edit(res.items)
    setAnnouncement(at(res.items, linkId, label, true, t))
  }

  const active = loaded.series.find((s) => s.status === 'active' && s.id !== v.id)
  const full = count(items) >= maxItems
  const found = problems(items)
  const open = translations.find((x) => x.language === editing)
  const refusedFor = translations.find((x) => x.language === incomplete)
  const missing = refusedFor ? progress(refusedFor, description, items).missing : []
  const missingIn = refusedFor ? t('questionnaire.missingIn', { language: contentLanguageNames[refusedFor.language] }) : undefined

  return (
    <Page wide>
      <div className="heading">
        <h1>{title.trim() || t('questionnaire.untitled')}</h1>
        <span className="tag draft">{t('questionnaire.statuses.draft')}</span>
      </div>
      {leavingTo !== undefined && (
        <Confirm
          question={t('questionnaire.leaveQuestion')}
          lines={[]}
          confirm={t('questionnaire.leave')}
          cancel={t('questionnaire.stay')}
          danger
          onConfirm={() => leave(leavingTo)}
          onCancel={() => setLeavingTo(undefined)}
        />
      )}
      <p className="sr-only" role="status" aria-live="polite">
        {announcement}
      </p>
      <div inert={publishing}>
        <Languages
          language={language}
          translations={translations}
          description={description}
          items={items}
          editing={open?.language}
          adding={adding}
          onLanguage={(l) => {
            setLanguage(l)
            changed()
          }}
          onEdit={setEditing}
          onAdding={setAdding}
          onAdd={addTranslation}
          onRemove={removeTranslation}
        />
      </div>
      {open ? (
        <div inert={publishing}>
          <TranslationEditor language={language} translation={open} title={title} description={description} items={items} onChange={(x) => editTranslations(translations.map((y) => (y.language === x.language ? x : y)))} />
        </div>
      ) : (
        <DragArea items={items} label={label} newLinkId={newLinkId} onChange={placed} onNested={() => setFailed({ message: 'questionnaire.nestedSection' })}>
          {(movingSection) => (
            <div className="builder" inert={publishing}>
              <Palette full={full} count={count(items)} onAdd={add} />
              <div className="sheet">
                <TextField
                  label={t('questionnaire.titleLabel')}
                  value={title}
                  max={maxTitle}
                  hint={t('questionnaire.templateHint')}
                  onChange={(x) => {
                    setTitle(x)
                    changed()
                  }}
                />
                <TextField
                  label={t('questionnaire.descriptionLabel')}
                  value={description}
                  max={maxDescription}
                  long
                  onChange={(x) => {
                    setDescription(x)
                    changed()
                  }}
                />
                <SortableList rows={rowsOf(items, movingSection)}>
                  <ul className="items">
                    {rowsOf(items, movingSection).map((r) => (
                      <SortableRow key={r.id} id={r.id} end={'end' in r}>
                        {(row) =>
                          'end' in r ? (
                            <li ref={row.ref} style={row.style} className="section-end" aria-hidden="true" />
                          ) : (
                            <ItemCard
                              item={r.item}
                              depth={r.depth}
                              fresh={fresh === r.item.linkId}
                              nodeRef={row.ref}
                              style={row.style}
                              handle={row.handle}
                              dragging={row.dragging}
                              label={label}
                              open={opened.includes(r.item.linkId)}
                              problems={found.get(r.item.linkId) ?? []}
                              missingIn={missing.includes(r.item.linkId) ? missingIn : undefined}
                              attention={flagged.includes(r.item.linkId) || found.has(r.item.linkId) || missing.includes(r.item.linkId)}
                              canUp={canStep(items, r.item.linkId, -1)}
                              canDown={canStep(items, r.item.linkId, 1)}
                              dependents={dependents(items, r.item.linkId)}
                              earlier={earlierQuestions(items, r.item.linkId)}
                              questionOf={(id) => find(items, id)}
                              onToggle={() =>
                                setOpened((o) => (o.includes(r.item.linkId) ? o.filter((x) => x !== r.item.linkId) : [...o, r.item.linkId]))
                              }
                              onStep={(dir) => stepItem(r.item.linkId, dir)}
                              onDelete={() => edit(remove(items, r.item.linkId))}
                              onChange={(patch) => edit(update(items, r.item.linkId, patch))}
                              onKind={(kind) => {
                                const next = newLinkId()
                                edit(changeKind(items, r.item.linkId, kind, next))
                                setOpened((o) => o.map((x) => (x === r.item.linkId ? next : x)))
                              }}
                            />
                          )
                        }
                      </SortableRow>
                    ))}
                  </ul>
                </SortableList>
                <ListEnd>
                  {(ref, over) => (
                    <div ref={ref} className={over ? 'list-end over' : 'list-end'}>
                      {items.length === 0 && <p className="muted">{t('questionnaire.empty')}</p>}
                    </div>
                  )}
                </ListEnd>
              </div>
            </div>
          )}
        </DragArea>
      )}
      <FailureAlert failure={failed} onReload={onReload} />
      {confirm === 'translate' && (
        <Confirm
          question={t('questionnaire.translateQuestion')}
          lines={[t('questionnaire.translateWarning')]}
          confirm={t('questionnaire.addTranslation')}
          cancel={t('questionnaire.publishWithout')}
          onConfirm={() => {
            setConfirm(undefined)
            setAdding(true)
          }}
          onCancel={() => setConfirm('publish')}
        />
      )}
      {confirm === 'publish' && (
        <Confirm
          question={t('questionnaire.publishQuestion')}
          lines={active ? [t('questionnaire.publishWarning'), t('questionnaire.publishRetires', { n: active.version })] : [t('questionnaire.publishWarning')]}
          confirm={t('questionnaire.publish')}
          cancel={t('questionnaire.cancel')}
          busy={busy || publishing}
          onConfirm={publish}
          onCancel={() => setConfirm(undefined)}
        />
      )}
      {confirm === 'discard' && (
        <Confirm
          question={t('questionnaire.discardQuestion')}
          lines={[t('questionnaire.discardWarning')]}
          confirm={t('questionnaire.discardConfirm')}
          cancel={t('questionnaire.cancel')}
          danger
          busy={busy}
          onConfirm={discard}
          onCancel={() => setConfirm(undefined)}
        />
      )}
      <div className="actions builder-actions">
        <button type="button" className="primary" disabled={busy} onClick={save}>
          {busy ? t('questionnaire.saving') : t('questionnaire.save')}
        </button>
        <button type="button" disabled={busy} onClick={askPublish}>
          {t('questionnaire.publish')}
        </button>
        <button type="button" className="quiet danger-text" disabled={busy} onClick={() => setConfirm('discard')}>
          {t('questionnaire.discard')}
        </button>
        {saved && !dirty && (
          <span className="muted" role="status">
            {t('questionnaire.saved')}
          </span>
        )}
      </div>
    </Page>
  )
}

function Palette({ full, count: n, onAdd }: { full: boolean; count: number; onAdd: (k: Kind) => void }) {
  const t = useT()
  return (
    <aside className="palette">
      <h2>{t('questionnaire.addItem')}</h2>
      <p className="hint">{t('questionnaire.addItemHint')}</p>
      <ul>
        {kinds.map((k) => (
          <li key={k}>
            <PaletteSource kind={k} disabled={full}>
              {(d) => (
                <button type="button" ref={d.ref} className={d.dragging ? 'kind dragging' : 'kind'} disabled={full} onClick={() => onAdd(k)} {...d.listeners}>
                  {t(`questionnaire.kinds.${k}`)}
                </button>
              )}
            </PaletteSource>
          </li>
        ))}
      </ul>
      <p className="hint">{t('questionnaire.itemCount', { n })}</p>
      {full && <p className="hint">{t('questionnaire.tooManyItems')}</p>}
    </aside>
  )
}

type PublishedProps = { api: ApiClient; loaded: Loaded; onChanged: (v: Version) => void }

// Published is an active or retired version: read only; an active one can be revised or retired.
function Published({ api, loaded, onChanged }: PublishedProps) {
  const t = useT()
  const label = labelOf(t)
  const v = loaded.version
  const [busy, setBusy] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [failed, setFailed] = useState<Failure>()
  const params = { params: { path: { id: v.id } } }

  async function revise() {
    setBusy(true)
    setFailed(undefined)
    const res = await attempt(api.POST('/questionnaires/{id}/revise', params))
    setBusy(false)
    if (res.ok) navigate(`/questionnaires/${encodeURIComponent(res.data.id)}`)
    else setFailed(failure(res.code, loaded.series))
  }

  async function retire() {
    setBusy(true)
    setFailed(undefined)
    const res = await attempt(api.POST('/questionnaires/{id}/retire', params))
    setBusy(false)
    setConfirming(false)
    if (res.ok) onChanged(fromJSON(res.data))
    else setFailed(failure(res.code, loaded.series))
  }

  return (
    <Page>
      <div className="heading">
        <h1>{v.title}</h1>
        <span className={`tag ${v.status}`}>{t(`questionnaire.statuses.${v.status}`)}</span>
      </div>
      {v.description && <p className="muted">{v.description}</p>}
      {v.status === 'active' ? (
        <>
          <p className="notice">{t('questionnaire.published')}</p>
          <div className="actions published-actions">
            <button type="button" className="primary" disabled={busy} onClick={revise}>
              {t('questionnaire.revise')}
            </button>
            <button type="button" disabled={busy} onClick={() => setConfirming(true)}>
              {t('questionnaire.retire')}
            </button>
          </div>
          {confirming && (
            <Confirm
              question={t('questionnaire.retireQuestion')}
              lines={[t('questionnaire.retireWarning')]}
              confirm={t('questionnaire.retire')}
              cancel={t('questionnaire.cancel')}
              danger
              busy={busy}
              onConfirm={retire}
              onCancel={() => setConfirming(false)}
            />
          )}
        </>
      ) : (
        <p className="notice">{t('questionnaire.retired')}</p>
      )}
      <FailureAlert failure={failed} />
      <ul className="items readonly">
        {v.items.flatMap((it) => [it, ...it.items]).map((it) => (
          <li key={it.linkId} className={['item', it.kind === 'group' ? 'section' : '', v.items.includes(it) ? '' : 'nested'].filter(Boolean).join(' ')}>
            <div className="item-head">
              <span className="item-text">{label(it)}</span>
              <span className="tag">{t(`questionnaire.kinds.${it.kind}`)}</span>
              <FilesSummary item={it} />
              {it.required && <span className="tag">{t('questionnaire.required')}</span>}
            </div>
          </li>
        ))}
      </ul>
    </Page>
  )
}

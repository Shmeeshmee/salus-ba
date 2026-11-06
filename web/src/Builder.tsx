import { useEffect, useRef, useState, type ReactNode } from 'react'
import type { ApiClient, ErrorCode } from './api/client'
import { FhirError, type Bundle, type FhirClient, type Questionnaire } from './api/fhir'
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
import { ItemCard, TextField } from './ItemCard'
import { useLoad } from './load'
import { questionnaires as t, somethingWrong } from './messages'
import { fromFHIR, fromJSON, toRequest, type Version } from './questionnaire'
import { guardLeaving, navigate } from './route'
import { Alert, Confirm, Link } from './ui'

type Loaded = { version: Version; series: Version[] }
type Props = { api: ApiClient; fhir: FhirClient; id: string }

const label = (it: Item) => it.text.trim() || t.kinds[it.kind]
const newLinkId = () => crypto.randomUUID()

// Builder is the page of one questionnaire version: a draft is edited (the palette, the items, save,
// publish, discard); an active version offers "Edit as new version" and "Retire"; a retired one is read only.
export function Builder({ api, fhir, id }: Props) {
  const { state, reload, set } = useLoad(async (): Promise<Loaded> => {
    const q = await fhir.get<Questionnaire>(`Questionnaire/${encodeURIComponent(id)}`)
    const bundle = await fhir.get<Bundle<Questionnaire>>(`Questionnaire?${new URLSearchParams({ url: q.url })}`)
    return { version: fromFHIR(q), series: (bundle.entry ?? []).map((e) => fromFHIR(e.resource)) }
  }, [fhir, id])

  if (state.name === 'loading') return <Page>{<p className="muted">Loading</p>}</Page>
  if (state.name === 'failed') {
    const gone = state.error instanceof FhirError && (state.error.status === 404 || state.error.status === 403)
    return <Page>{<Alert message={gone ? t.notAvailable : somethingWrong} />}</Page>
  }
  const loaded = state.value
  const changed = (version: Version) => set({ ...loaded, version })
  if (loaded.version.status === 'draft') return <Editor api={api} loaded={loaded} onReload={reload} onPublished={changed} />
  return <Published api={api} loaded={loaded} onChanged={changed} />
}

function Page({ children, wide }: { children: ReactNode; wide?: boolean }) {
  return (
    <section className={wide ? 'panel builder-page' : 'panel'}>
      <Link to="/questionnaires" className="back">
        <ChevronIcon direction="left" />
        {t.title}
      </Link>
      {children}
    </section>
  )
}

type Failure = { message: string; reload?: boolean; draft?: string }

// failure maps an API refusal to its W29 text.
function failure(code: ErrorCode | undefined, series: Version[]): Failure {
  switch (code) {
    case 'version_conflict':
      return { message: t.changed, reload: true }
    case 'not_draft':
      return { message: t.notDraft }
    case 'invalid_questionnaire':
      return { message: t.needsAttention }
    case 'not_found':
      return { message: t.notAvailable }
    case 'draft_exists':
      return { message: t.draftExists, draft: series.find((v) => v.status === 'draft')?.id }
    default:
      return { message: somethingWrong }
  }
}

function FailureAlert({ failure, onReload }: { failure?: Failure; onReload?: () => void }) {
  if (!failure) return null
  return (
    <div className="failure">
      <Alert message={failure.message} />
      {failure.reload && onReload && (
        <div className="actions">
          <button type="button" onClick={onReload}>
            {t.reload}
          </button>
        </div>
      )}
      {failure.draft && (
        <div className="actions">
          <button type="button" onClick={() => navigate(`/questionnaires/${encodeURIComponent(failure.draft ?? '')}`)}>
            {t.openDraft}
          </button>
        </div>
      )}
    </div>
  )
}

type EditorProps = { api: ApiClient; loaded: Loaded; onReload: () => void; onPublished: (v: Version) => void }

function Editor({ api, loaded, onReload, onPublished }: EditorProps) {
  const [title, setTitle] = useState(loaded.version.title)
  const [description, setDescription] = useState(loaded.version.description)
  const [items, setItems] = useState(loaded.version.items)
  const [versionId, setVersionId] = useState(loaded.version.versionId)
  const [dirty, setDirty] = useState(false)
  const [saved, setSaved] = useState(false)
  const [busy, setBusy] = useState(false)
  const [publishing, setPublishing] = useState(false)
  const [opened, setOpened] = useState<string[]>([])
  const [flagged, setFlagged] = useState<string[]>([])
  const [failed, setFailed] = useState<Failure>()
  const [confirm, setConfirm] = useState<'publish' | 'discard'>()
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

  async function save(): Promise<number | undefined> {
    setBusy(true)
    setFailed(undefined)
    setSaved(false)
    const sent = edits.current
    const res = await attempt(
      api.PUT('/questionnaires/{id}', {
        params: { path: { id: v.id }, header: { 'If-Match': `W/"${versionId}"` } },
        body: toRequest(title, description, items),
      }),
    )
    setBusy(false)
    if (!res.ok) {
      refused(res)
      return undefined
    }
    setVersionId(res.data.versionId)
    setFlagged([])
    if (edits.current === sent) {
      setDirty(false)
      dirtyRef.current = false
      setSaved(true)
    }
    return res.data.versionId
  }

  function refused(res: Outcome<unknown>) {
    if (res.ok) return
    setFailed(failure(res.code, loaded.series))
    if (res.code !== 'invalid_questionnaire') return
    const ids = res.linkIds ?? []
    setFlagged(ids)
    setOpened((o) => [...new Set([...o, ...ids])])
  }

  // askPublish checks the items as the API will before it asks: a draft with problems is shown, not offered.
  function askPublish() {
    const ids = [...found.keys()]
    if (ids.length === 0) {
      setConfirm('publish')
      return
    }
    setFailed({ message: t.needsAttention })
    setOpened((o) => [...new Set([...o, ...ids])])
  }

  // publish saves what is unsaved, then publishes that version. The items are out of reach meanwhile (inert), so
  // no edit can come after the save and be lost to the published version.
  async function publish() {
    setPublishing(true)
    const current = dirty ? await save() : versionId
    if (current === undefined) {
      setPublishing(false)
      setConfirm(undefined)
      return
    }
    setBusy(true)
    const res = await attempt(
      api.POST('/questionnaires/{id}/publish', { params: { path: { id: v.id }, header: { 'If-Match': `W/"${current}"` } } }),
    )
    setBusy(false)
    setPublishing(false)
    setConfirm(undefined)
    if (res.ok) onPublished(fromJSON(res.data))
    else refused(res)
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
    setAnnouncement(at(res.items, linkId, label, true))
  }

  const active = loaded.series.find((s) => s.status === 'active' && s.id !== v.id)
  const full = count(items) >= maxItems
  const found = problems(items)

  return (
    <Page wide>
      <div className="heading">
        <h1>{title.trim() || t.untitled}</h1>
        <span className="tag draft">{t.statuses.draft}</span>
      </div>
      {leavingTo !== undefined && (
        <Confirm
          question={t.leaveQuestion}
          lines={[]}
          confirm={t.leave}
          cancel={t.stay}
          danger
          onConfirm={() => leave(leavingTo)}
          onCancel={() => setLeavingTo(undefined)}
        />
      )}
      <p className="sr-only" role="status" aria-live="polite">
        {announcement}
      </p>
      <DragArea items={items} label={label} newLinkId={newLinkId} onChange={placed} onNested={() => setFailed({ message: t.nestedSection })}>
        {(movingSection) => (
          <div className="builder" inert={publishing}>
            <Palette full={full} count={count(items)} onAdd={add} />
            <div className="sheet">
              <TextField
                label={t.titleLabel}
                value={title}
                max={maxTitle}
                hint={t.templateHint}
                onChange={(x) => {
                  setTitle(x)
                  changed()
                }}
              />
              <TextField
                label={t.descriptionLabel}
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
                            attention={flagged.includes(r.item.linkId) || found.has(r.item.linkId)}
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
                    {items.length === 0 && <p className="muted">{t.empty}</p>}
                  </div>
                )}
              </ListEnd>
            </div>
          </div>
        )}
      </DragArea>
      <FailureAlert failure={failed} onReload={onReload} />
      {confirm === 'publish' && (
        <Confirm
          question={t.publishQuestion}
          lines={active ? [t.publishWarning, t.publishRetires(active.version)] : [t.publishWarning]}
          confirm={t.publish}
          cancel={t.cancel}
          busy={busy || publishing}
          onConfirm={publish}
          onCancel={() => setConfirm(undefined)}
        />
      )}
      {confirm === 'discard' && (
        <Confirm
          question={t.discardQuestion}
          lines={[t.discardWarning]}
          confirm={t.discardConfirm}
          cancel={t.cancel}
          danger
          busy={busy}
          onConfirm={discard}
          onCancel={() => setConfirm(undefined)}
        />
      )}
      <div className="actions builder-actions">
        <button type="button" className="primary" disabled={busy} onClick={save}>
          {busy ? t.saving : t.save}
        </button>
        <button type="button" disabled={busy} onClick={askPublish}>
          {t.publish}
        </button>
        <button type="button" className="quiet danger-text" disabled={busy} onClick={() => setConfirm('discard')}>
          {t.discard}
        </button>
        {saved && !dirty && (
          <span className="muted" role="status">
            {t.saved}
          </span>
        )}
      </div>
    </Page>
  )
}

function Palette({ full, count: n, onAdd }: { full: boolean; count: number; onAdd: (k: Kind) => void }) {
  return (
    <aside className="palette">
      <h2>{t.addItem}</h2>
      <p className="hint">{t.addItemHint}</p>
      <ul>
        {kinds.map((k) => (
          <li key={k}>
            <PaletteSource kind={k} disabled={full}>
              {(d) => (
                <button type="button" ref={d.ref} className={d.dragging ? 'kind dragging' : 'kind'} disabled={full} onClick={() => onAdd(k)} {...d.listeners}>
                  {t.kinds[k]}
                </button>
              )}
            </PaletteSource>
          </li>
        ))}
      </ul>
      <p className="hint">{t.itemCount(n)}</p>
      {full && <p className="hint">{t.tooManyItems}</p>}
    </aside>
  )
}

type PublishedProps = { api: ApiClient; loaded: Loaded; onChanged: (v: Version) => void }

// Published is an active or retired version: read only; an active one can be revised or retired.
function Published({ api, loaded, onChanged }: PublishedProps) {
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
        <span className={`tag ${v.status}`}>{t.statuses[v.status]}</span>
      </div>
      {v.description && <p className="muted">{v.description}</p>}
      {v.status === 'active' ? (
        <>
          <p className="notice">{t.published}</p>
          <div className="actions">
            <button type="button" className="primary" disabled={busy} onClick={revise}>
              {t.revise}
            </button>
            <button type="button" disabled={busy} onClick={() => setConfirming(true)}>
              {t.retire}
            </button>
          </div>
          {confirming && (
            <Confirm
              question={t.retireQuestion}
              lines={[t.retireWarning]}
              confirm={t.retire}
              cancel={t.cancel}
              danger
              busy={busy}
              onConfirm={retire}
              onCancel={() => setConfirming(false)}
            />
          )}
        </>
      ) : (
        <p className="notice">{t.retired}</p>
      )}
      <FailureAlert failure={failed} />
      <ul className="items readonly">
        {v.items.flatMap((it) => [it, ...it.items]).map((it) => (
          <li key={it.linkId} className={['item', it.kind === 'group' ? 'section' : '', v.items.includes(it) ? '' : 'nested'].filter(Boolean).join(' ')}>
            <div className="item-head">
              <span className="item-text">{label(it)}</span>
              <span className="tag">{t.kinds[it.kind]}</span>
              {it.required && <span className="tag">{t.required}</span>}
            </div>
          </li>
        ))}
      </ul>
    </Page>
  )
}

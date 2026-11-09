import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  TouchSensor,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type Announcements,
  type CollisionDetection,
  type DragEndEvent,
  type UniqueIdentifier,
} from '@dnd-kit/core'
import { SortableContext, sortableKeyboardCoordinates, useSortable, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { useRef, useState, type CSSProperties, type PointerEvent, type ReactNode, type Ref } from 'react'
import { at, cancelled, find, insert, move, newItem, rowsOf, type Item, type Kind, type Result, type Row } from './builder'
import { useT } from './i18n/useT'

// The builder's drag and drop: the only module that imports @dnd-kit (an oxlint rule keeps it so). It maps
// @dnd-kit's events onto builder.ts and sets every text @dnd-kit gives a screen reader to COPY W41 and W42.

export const listEnd = 'questionnaire-end' // the drop area after the last row
const fromPalette = 'palette-'

// MouseSensor takes mouse and pen; touch goes through the TouchSensor, whose delay leaves the page scrollable.
class MouseSensor extends PointerSensor {
  static activators = [
    {
      eventName: 'onPointerDown' as const,
      handler: ({ nativeEvent: e }: PointerEvent) => e.pointerType !== 'touch' && e.isPrimary && e.button === 0,
    },
  ]
}

// closestTop picks the row whose top is nearest the dragged item's top. @dnd-kit's keyboard moves line the
// tops up, and an open card is much taller than the closed ones: by centre or corners it stays nearest its
// own place and a keyboard move up never lands.
const closestTop: CollisionDetection = ({ collisionRect, droppableRects, droppableContainers }) =>
  droppableContainers
    .flatMap((c) => {
      const r = droppableRects.get(c.id)
      return r ? [{ id: c.id, data: { droppableContainer: c, value: Math.abs(r.top - collisionRect.top) } }] : []
    })
    .sort((a, b) => a.data.value - b.data.value)

type Projection = { result: Result; id: string }

// project is what a drop of active over over would do: a palette type is inserted, an item moved.
function project(items: Item[], active: UniqueIdentifier, over: UniqueIdentifier | undefined, linkId: string): Projection | undefined {
  if (over === undefined) return undefined
  const a = String(active)
  const o = String(over)
  if (a.startsWith(fromPalette)) {
    const it = newItem(a.slice(fromPalette.length) as Kind, linkId)
    return { result: insert(items, it, o === listEnd ? undefined : o), id: it.linkId }
  }
  const dragged = find(items, a)
  const last = rowsOf(items, dragged?.kind === 'group').at(-1)?.id ?? a
  return { result: move(items, a, o === listEnd ? last : o), id: a }
}

type Label = (it: Item) => string

type AreaProps = {
  items: Item[]
  label: Label
  newLinkId: () => string
  onChange: (items: Item[], placed: string) => void
  onNested: () => void
  children: (movingSection: boolean) => ReactNode
}

// DragArea is the drag-and-drop context of the palette and the list. While a section is moved, children
// get true and list the top level only.
export function DragArea({ items, label, newLinkId, onChange, onNested, children }: AreaProps) {
  const t = useT()
  const [active, setActive] = useState<string>()
  const lastOver = useRef<UniqueIdentifier>(undefined) // the drop place last announced
  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 5 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 250, tolerance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )
  const name = (id: UniqueIdentifier) => {
    const s = String(id)
    if (s.startsWith(fromPalette)) return t(`questionnaire.kinds.${s.slice(fromPalette.length) as Kind}`)
    const it = find(items, s)
    return it ? label(it) : ''
  }
  const placed = (p: Projection | undefined) => {
    if (!p) return undefined
    return p.result.ok ? at(p.result.items, p.id, label, false, t) : t('questionnaire.nestedSection')
  }
  const fromList = (id: UniqueIdentifier) => !String(id).startsWith(fromPalette)

  const announcements: Announcements = {
    onDragStart: ({ active }) => {
      lastOver.current = active.id // a drag starts over its own place: "Moving" is said, not the place
      return t('questionnaire.moving', { item: name(active.id) })
    },
    onDragOver: ({ active, over }) => {
      if (over?.id === lastOver.current) return undefined
      lastOver.current = over?.id
      return placed(project(items, active.id, over?.id, 'new'))
    },
    onDragEnd: ({ active, over }) => {
      const p = project(items, active.id, over?.id, 'new')
      if (p?.result.ok) return at(p.result.items, p.id, label, true, t)
      return fromList(active.id) ? cancelled(items, String(active.id), label, t) : undefined
    },
    onDragCancel: ({ active }) => (fromList(active.id) ? cancelled(items, String(active.id), label, t) : undefined),
  }

  function onDragEnd({ active, over }: DragEndEvent) {
    setActive(undefined)
    const p = project(items, active.id, over?.id, newLinkId())
    if (!p) return
    if (p.result.ok) onChange(p.result.items, p.id)
    else onNested()
  }

  const movingSection = active !== undefined && find(items, active)?.kind === 'group'
  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestTop}
      accessibility={{ announcements, screenReaderInstructions: { draggable: t('questionnaire.dragInstructions') } }}
      onDragStart={({ active }) => setActive(String(active.id))}
      onDragEnd={onDragEnd}
      onDragCancel={() => setActive(undefined)}
    >
      {children(movingSection)}
      {/* A type from the palette is dragged as a copy: the palette stays put, so releasing the pointer over the
          list is not also a click on the palette's button (which would add the type a second time). Items of
          the list move themselves; with an overlay mounted @dnd-kit would measure the keyboard's moves on it. */}
      {active?.startsWith(fromPalette) && (
        <DragOverlay dropAnimation={null}>
          <div className="kind overlay">{name(active)}</div>
        </DragOverlay>
      )}
    </DndContext>
  )
}

export function SortableList({ rows, children }: { rows: Row[]; children: ReactNode }) {
  return (
    <SortableContext items={rows.map((r) => r.id)} strategy={verticalListSortingStrategy}>
      {children}
    </SortableContext>
  )
}

type Handle = Record<string, unknown> & { ref: (el: HTMLElement | null) => void }
type RowParts = { ref: (el: HTMLElement | null) => void; style: CSSProperties; handle: Handle; dragging: boolean }

// SortableRow is one row of the list: an item, dragged by its handle, or a section's end, which only takes
// drops. @dnd-kit's own role description is left out, so no English text of its own is read.
export function SortableRow({ id, end, children }: { id: string; end: boolean; children: (row: RowParts) => ReactNode }) {
  const s = useSortable({ id, disabled: end ? { draggable: true } : false })
  const { 'aria-roledescription': _, ...attributes } = s.attributes
  const style: CSSProperties = {
    transform: s.transform ? `translate3d(${Math.round(s.transform.x)}px, ${Math.round(s.transform.y)}px, 0)` : undefined,
    transition: s.transition,
  }
  return children({ ref: s.setNodeRef, style, handle: { ref: s.setActivatorNodeRef, ...attributes, ...s.listeners }, dragging: s.isDragging })
}

type SourceParts = { ref: Ref<HTMLButtonElement>; listeners: Record<string, unknown>; dragging: boolean }

// PaletteSource lets a type be dragged into the list with a pointer or touch, as a copy (DragArea's overlay).
// From the keyboard the type's button adds it at the end, so the keyboard sensor is not attached here.
export function PaletteSource({ kind, disabled, children }: { kind: Kind; disabled: boolean; children: (p: SourceParts) => ReactNode }) {
  const d = useDraggable({ id: `${fromPalette}${kind}`, disabled })
  const { onKeyDown: _, ...listeners } = d.listeners ?? {}
  return children({ ref: d.setNodeRef, listeners, dragging: d.isDragging })
}

// ListEnd is the drop area after the last row (all of the list while it is empty).
export function ListEnd({ children }: { children: (ref: (el: HTMLElement | null) => void, over: boolean) => ReactNode }) {
  const d = useDroppable({ id: listEnd })
  return children(d.setNodeRef, d.isOver)
}

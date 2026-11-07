import { act, cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { newApiClient } from './api/client'
import type { Attached } from './attachment'
import { Files } from './Files'
import { apiError, fakeFetch, paths, type Route } from './test/fakeFetch'

afterEach(cleanup)

// Synthetic files of each kind; the titles are placeholders, never a real file's name.
const photo: Attached = { id: 'b0000000-0000-4000-8000-0000000000b1', kind: 'photo', contentType: 'image/jpeg', title: 'placeholder-photo.jpg', size: 2048 }
const pdf: Attached = { id: 'b0000000-0000-4000-8000-0000000000b2', kind: 'document', contentType: 'application/pdf', title: 'placeholder.pdf', size: 1_500_000 }
const video: Attached = { id: 'b0000000-0000-4000-8000-0000000000b3', kind: 'video', contentType: 'video/mp4', title: 'placeholder.mp4', size: 42_000_000 }
const content = (f: Attached) => `${location.origin}/binaries/${f.id}/content`
const status = (f: Attached) => `GET /binaries/${f.id}`

function renderFiles(files: Attached[], routes: Record<string, Route> = {}) {
  const fake = fakeFetch(routes)
  render(
    <>
      <button type="button">Before</button>
      <Files api={newApiClient(fake.fetch)} files={files} />
    </>,
  )
  return fake
}

const viewer = () => screen.getByRole('dialog')
const rowOf = (title: string) => within(screen.getByText(title).closest('li') as HTMLElement)

describe('Files: the list (W8, W9)', () => {
  it('lists each file with its title, kind and size, to open or download', () => {
    renderFiles([photo, pdf, video])

    expect(screen.getAllByRole('listitem').map((li) => li.querySelector('.file-details')?.textContent)).toEqual([
      'Photo, 2 KB',
      'Document, 1.5 MB',
      'Video, 42.0 MB',
    ])
    const download = rowOf('placeholder.pdf').getByRole('link', { name: 'Download' }) as HTMLAnchorElement
    expect(download.href).toBe(content(pdf))
    expect(download.getAttribute('download')).toBe('placeholder.pdf')
    expect(rowOf('placeholder.pdf').getByRole('button', { name: 'Open' })).toBeTruthy()
  })

  it('lists a file whose url names no binary without actions', () => {
    renderFiles([{ ...photo, id: undefined }])

    expect(screen.getByText('placeholder-photo.jpg')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Open' })).toBeNull()
    expect(screen.queryByRole('link', { name: 'Download' })).toBeNull()
  })

  it('names a file without a title by its kind', () => {
    renderFiles([{ ...photo, title: '' }])
    expect(screen.getByText('Photo', { selector: '.file-title' })).toBeTruthy()
  })
})

describe('Files: the viewer (W10, W11)', () => {
  it('shows a photo, takes the focus, and gives it back to Open on Close', () => {
    renderFiles([photo, pdf])
    const open = rowOf('placeholder-photo.jpg').getByRole('button', { name: 'Open' })
    open.focus()
    fireEvent.click(open)

    const d = viewer()
    expect(d.getAttribute('aria-modal')).toBe('true')
    expect(d.contains(document.activeElement)).toBe(true)
    expect(within(d).getByText('File 1 of 2')).toBeTruthy()
    const img = within(d).getByRole('img', { name: 'Photo: placeholder-photo.jpg' }) as HTMLImageElement
    expect(img.src).toBe(content(photo))

    fireEvent.click(within(d).getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(document.activeElement).toBe(open)
  })

  it('shows a PDF in the browser\'s own viewer, with Download when it cannot (W12)', () => {
    renderFiles([pdf])
    fireEvent.click(screen.getByRole('button', { name: 'Open' }))

    const object = viewer().querySelector('object') as HTMLObjectElement
    expect(object.getAttribute('data')).toBe(content(pdf))
    expect(object.getAttribute('type')).toBe('application/pdf')
    expect(object.getAttribute('aria-label')).toBe('Document: placeholder.pdf')
    expect(within(object).getByText('This file cannot be shown here.')).toBeTruthy()
    expect((within(object).getByRole('link', { name: 'Download' }) as HTMLAnchorElement).href).toBe(content(pdf))
  })

  it('plays a video from the content url, so the browser seeks with Range requests', () => {
    renderFiles([video])
    fireEvent.click(screen.getByRole('button', { name: 'Open' }))

    const v = viewer().querySelector('video') as HTMLVideoElement
    expect(v.getAttribute('src')).toBe(content(video))
    expect(v.controls).toBe(true)
    expect(v.getAttribute('preload')).toBe('metadata')
    expect(v.getAttribute('aria-label')).toBe('Video: placeholder.mp4')
  })

  it('steps through the files with Previous file and Next file, and marks the ends', () => {
    renderFiles([photo, pdf, video])
    fireEvent.click(rowOf('placeholder.pdf').getByRole('button', { name: 'Open' }))
    const d = viewer()
    const previous = within(d).getByRole('button', { name: 'Previous file' })
    const next = within(d).getByRole('button', { name: 'Next file' })

    expect(within(d).getByText('File 2 of 3')).toBeTruthy()
    fireEvent.click(next)
    expect(within(d).getByText('File 3 of 3')).toBeTruthy()
    expect(d.querySelector('video')).toBeTruthy()
    expect(next.getAttribute('aria-disabled')).toBe('true')
    fireEvent.click(next)
    expect(within(d).getByText('File 3 of 3')).toBeTruthy()

    fireEvent.click(previous)
    fireEvent.click(previous)
    expect(within(d).getByText('File 1 of 3')).toBeTruthy()
    expect(within(d).getByRole('img', { name: 'Photo: placeholder-photo.jpg' })).toBeTruthy()
    expect(previous.getAttribute('aria-disabled')).toBe('true')
  })

  it('offers no Previous file and Next file for a single file', () => {
    renderFiles([photo])
    fireEvent.click(screen.getByRole('button', { name: 'Open' }))
    expect(within(viewer()).queryByRole('button', { name: 'Next file' })).toBeNull()
  })

  it('closes on escape and keeps the focus inside while open', () => {
    renderFiles([photo, pdf])
    const open = rowOf('placeholder-photo.jpg').getByRole('button', { name: 'Open' })
    open.focus()
    fireEvent.click(open)
    const d = viewer()
    const focusable = Array.from(d.querySelectorAll<HTMLElement>('button, a[href]'))
    const first = focusable[0]
    const last = focusable.at(-1) as HTMLElement

    last.focus()
    fireEvent.keyDown(last, { key: 'Tab' })
    expect(document.activeElement).toBe(first)
    fireEvent.keyDown(first, { key: 'Tab', shiftKey: true })
    expect(document.activeElement).toBe(last)

    fireEvent.keyDown(document.activeElement as HTMLElement, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(document.activeElement).toBe(open)
  })
})

describe('Files: failures (W12, W13)', () => {
  it('says a file that is gone is no longer available (404)', async () => {
    const { requests } = renderFiles([photo], { [status(photo)]: () => apiError(404, 'not_found') })
    fireEvent.click(screen.getByRole('button', { name: 'Open' }))
    fireEvent.error(within(viewer()).getByRole('img'))

    expect((await within(viewer()).findByRole('alert')).textContent).toBe('This file is no longer available.')
    expect(within(viewer()).queryByRole('img')).toBeNull()
    expect(within(viewer()).queryByRole('button', { name: 'Try again' })).toBeNull()
    expect(paths(requests)).toEqual([status(photo)])
  })

  it('says any other failure could not be loaded, and tries again', async () => {
    renderFiles([photo], { [status(photo)]: () => apiError(500, 'internal error') })
    fireEvent.click(screen.getByRole('button', { name: 'Open' }))
    fireEvent.error(within(viewer()).getByRole('img'))

    expect((await within(viewer()).findByRole('alert')).textContent).toBe('The file could not be loaded.')
    fireEvent.click(within(viewer()).getByRole('button', { name: 'Try again' }))
    expect(within(viewer()).getByRole('img', { name: 'Photo: placeholder-photo.jpg' })).toBeTruthy()
    expect(within(viewer()).queryByRole('alert')).toBeNull()
  })

  it('says a file still being checked could not be loaded', async () => {
    renderFiles([video], { [status(video)]: () => Response.json({ ...video, status: 'quarantined' }) })
    fireEvent.click(screen.getByRole('button', { name: 'Open' }))
    fireEvent.error(viewer().querySelector('video') as HTMLVideoElement)

    expect((await within(viewer()).findByRole('alert')).textContent).toBe('The file could not be loaded.')
  })

  it('offers Download when the file is there but the browser cannot show it', async () => {
    renderFiles([photo], { [status(photo)]: () => Response.json({ ...photo, status: 'clean' }) })
    fireEvent.click(screen.getByRole('button', { name: 'Open' }))
    fireEvent.error(within(viewer()).getByRole('img'))

    expect(await within(viewer()).findByText('This file cannot be shown here.')).toBeTruthy()
    expect((within(viewer()).getByRole('link', { name: 'Download' }) as HTMLAnchorElement).href).toBe(content(photo))
  })

  it('ignores the answer about a file the viewer has moved on from', async () => {
    let answer: (r: Response) => void = () => {}
    renderFiles([photo, pdf], { [status(photo)]: () => new Promise<Response>((resolve) => (answer = resolve)) })
    fireEvent.click(rowOf('placeholder-photo.jpg').getByRole('button', { name: 'Open' }))
    fireEvent.error(within(viewer()).getByRole('img'))
    fireEvent.click(within(viewer()).getByRole('button', { name: 'Next file' }))

    await act(async () => answer(apiError(404, 'not_found')))
    await vi.waitFor(() => expect(within(viewer()).getByText('File 2 of 2')).toBeTruthy())
    expect(within(viewer()).queryByRole('alert')).toBeNull()
    expect(viewer().querySelector('object')).toBeTruthy()
  })
})

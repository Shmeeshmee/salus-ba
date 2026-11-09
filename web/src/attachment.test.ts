import { describe, expect, it } from 'vitest'
import { binaryContentURL } from './api/client'
import { filesOf, sizeText } from './attachment'
import { translator } from './test/translate'

const id = 'b0000000-0000-4000-8000-0000000000b1'
const url = `https://api.example.test/binaries/${id}/content`

describe('filesOf', () => {
  it('reads each attachment answer: its id from the url, its kind from the content type', () => {
    const files = filesOf([
      { valueAttachment: { contentType: 'image/jpeg', size: 2048, title: 'placeholder-photo.jpg', hash: 'AAAA', url } },
      { valueAttachment: { contentType: 'image/png', size: 10, title: 'placeholder.png', url } },
      { valueAttachment: { contentType: 'application/pdf', size: 1_500_000, title: 'placeholder.pdf', url } },
      { valueAttachment: { contentType: 'video/mp4', size: 42_000_000, title: 'placeholder.mp4', url } },
    ])
    expect(files.map((f) => [f.id, f.kind, f.title, f.size])).toEqual([
      [id, 'photo', 'placeholder-photo.jpg', 2048],
      [id, 'photo', 'placeholder.png', 10],
      [id, 'document', 'placeholder.pdf', 1_500_000],
      [id, 'video', 'placeholder.mp4', 42_000_000],
    ])
  })

  it('leaves the id out when the url is not a binary content url, and skips answers that are not files', () => {
    const files = filesOf([
      { valueString: 'Placeholder' },
      { valueAttachment: { contentType: 'image/jpeg', size: 1, title: 'a', url: 'https://elsewhere.example.test/x.jpg' } },
      { valueAttachment: { contentType: 'image/jpeg', size: 1, title: 'b', url: `https://api.example.test/binaries/${id}/content/../../x` } },
      { valueAttachment: { contentType: 'image/jpeg', size: 1, title: 'c' } },
    ])
    expect(files.map((f) => [f.title, f.id])).toEqual([
      ['a', undefined],
      ['b', undefined],
      ['c', undefined],
    ])
  })
})

describe('sizeText (W8)', () => {
  it.each([
    [0, '1 KB'],
    [400, '1 KB'],
    [2048, '2 KB'],
    [999_000, '999 KB'],
    [999_600, '1.0 MB'],
    [1_500_000, '1.5 MB'],
    [20_000_000, '20.0 MB'],
    [199_960_000, '200.0 MB'],
  ])('writes %d bytes as %s', (bytes, text) => {
    expect(sizeText(bytes, translator('en'), 'en')).toBe(text)
  })

  it('writes the decimal the way the UI language does', () => {
    expect(sizeText(1_500_000, translator('hr'), 'hr')).toBe('1,5 MB')
  })
})

describe('binaryContentURL', () => {
  it("is the content endpoint on the page's own origin, the id encoded", () => {
    expect(binaryContentURL(id)).toBe(`${location.origin}/binaries/${id}/content`)
    expect(binaryContentURL('a/b')).toBe(`${location.origin}/binaries/a%2Fb/content`)
  })
})

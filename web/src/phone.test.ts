import { describe, expect, it } from 'vitest'
import { groupPhone } from './phone'

describe('groupPhone', () => {
  it.each([
    ['+38761000399', '+387 61 000 399'],
    ['+38733123456', '+387 33 123 456'],
    ['+387610003991', '+387 61 000 3991'],
    ['+4915112345678', '+4915112345678'],
    ['+38761', '+387 61'],
    ['', ''],
    ['061 000 399', '061 000 399'],
  ])('groups %s as %s', (raw, want) => {
    expect(groupPhone(raw)).toBe(want)
  })
})

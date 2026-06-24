import type { ChangeEvent } from 'react'

export function idempotencyKey() {
  return crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export function maskPhoneInput(event: ChangeEvent<HTMLInputElement>) {
  const digits = event.currentTarget.value.replace(/\D/g, '').slice(0, 13)
  if (!digits) {
    event.currentTarget.value = ''
    return
  }
  const withCountry = digits.startsWith('55') ? digits : `55${digits}`.slice(0, 13)
  const country = withCountry.slice(0, 2)
  const area = withCountry.slice(2, 4)
  const number = withCountry.slice(4)
  const first = number.length > 8 ? number.slice(0, 5) : number.slice(0, 4)
  const second = number.length > 8 ? number.slice(5, 9) : number.slice(4, 8)
  event.currentTarget.value = `+${country}${area ? ` (${area}` : ''}${area.length === 2 ? ')' : ''}${first ? ` ${first}` : ''}${second ? `-${second}` : ''}`
}

export const PRODUCT_SLOT_STATUSES = [
  { value: 'open', label: '可预约', type: 'success' },
  { value: 'occupied', label: '已占用', type: 'warning' },
  { value: 'locked', label: '已锁定', type: 'info' },
] as const

export const SLOT_MINUTES = 30
export const SLOT_MAX_ADVANCE_DAYS = 30

export function productSlotStatusLabel(value: string): string {
  return PRODUCT_SLOT_STATUSES.find((s) => s.value === value)?.label ?? value
}

export function productSlotStatusType(value: string): string {
  return PRODUCT_SLOT_STATUSES.find((s) => s.value === value)?.type ?? 'info'
}

// Formats a Date as local "YYYY-MM-DDTHH:mm" for datetime-local inputs.
export function toLocalInputValue(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// Formats a Date as an RFC3339 string with the local timezone offset.
export function toRFC3339Local(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  const offsetMin = -d.getTimezoneOffset()
  const sign = offsetMin >= 0 ? '+' : '-'
  const oh = pad(Math.floor(Math.abs(offsetMin) / 60))
  const om = pad(Math.abs(offsetMin) % 60)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:00${sign}${oh}:${om}`
}

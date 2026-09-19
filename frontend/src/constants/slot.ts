import type { SlotStatus } from '../types'

export const SLOT_STATUSES = [
  { value: 'available', label: '可预约', type: 'success' },
  { value: 'locked', label: '已预约', type: 'warning' },
  { value: 'released', label: '已释放', type: 'info' },
] as const

export function slotStatusLabel(value: string): string {
  return SLOT_STATUSES.find((s) => s.value === value)?.label ?? value
}

export function slotStatusType(value: SlotStatus | string): string {
  return SLOT_STATUSES.find((s) => s.value === value)?.type ?? 'info'
}

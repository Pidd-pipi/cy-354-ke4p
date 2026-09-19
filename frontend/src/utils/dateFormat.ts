export function formatDateTime(value?: string | number | Date): string {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function formatTime(value?: string | number | Date): string {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// Renders a half-hour meetup slot as "MM-DD HH:mm-HH:mm".
export function formatSlotRange(startValue?: string | number | Date | null): string {
  if (!startValue) return '-'
  const start = new Date(startValue)
  if (Number.isNaN(start.getTime())) return '-'
  const end = new Date(start.getTime() + 30 * 60 * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  const datePart = `${pad(start.getMonth() + 1)}-${pad(start.getDate())}`
  return `${datePart} ${pad(start.getHours())}:${pad(start.getMinutes())}-${pad(end.getHours())}:${pad(end.getMinutes())}`
}

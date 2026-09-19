export function formatDateTime(value?: string | number | Date | null): string {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function formatTime(value?: string | number | Date | null): string {
  if (!value) return '--:--'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '--:--'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// formatSlotRange renders a handover window like "09-21 14:00-14:30".
export function formatSlotRange(start?: string | null, end?: string | null): string {
  if (!start) return '-'
  const d = new Date(start)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  const day = `${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  return `${day} ${formatTime(start)}-${formatTime(end)}`
}

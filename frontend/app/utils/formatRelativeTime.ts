/** Format a date as relative time (e.g. "19 hours ago", "1 day ago"). */
export function formatRelativeTime(value: string | Date | null | undefined): string {
  if (!value) return ''
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)

  const seconds = Math.floor((Date.now() - date.getTime()) / 1000)
  if (seconds < 60) return seconds <= 1 ? 'just now' : `${seconds} seconds ago`

  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return minutes === 1 ? '1 minute ago' : `${minutes} minutes ago`

  const hours = Math.floor(minutes / 60)
  if (hours < 24) return hours === 1 ? '1 hour ago' : `${hours} hours ago`

  const days = Math.floor(hours / 24)
  if (days < 30) return days === 1 ? '1 day ago' : `${days} days ago`

  const months = Math.floor(days / 30)
  if (months < 12) return months === 1 ? '1 month ago' : `${months} months ago`

  const years = Math.floor(months / 12)
  return years === 1 ? '1 year ago' : `${years} years ago`
}

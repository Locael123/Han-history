const CN_NUM = ['〇', '一', '二', '三', '四', '五', '六', '七', '八', '九']

/** 1912 → 一九一二（用于报头与年份大数字下方的小字） */
export function chineseYear(year: number): string {
  if (!year) return ''
  return String(year)
    .split('')
    .map((ch) => CN_NUM[Number(ch)] ?? ch)
    .join('')
}

export function pad(n: number): string {
  return String(n).padStart(2, '0')
}

/** 1912.01.01 / 1912.01 / 1912 */
export function shortDate(year: number, month: number, day: number): string {
  if (!year) return '年份未详'
  if (!month) return `${year}`
  if (!day) return `${year}.${pad(month)}`
  return `${year}.${pad(month)}.${pad(day)}`
}

/** 一九一二年一月一日 */
export function longDate(year: number, month: number, day: number): string {
  if (!year) return '年份未详'
  let text = `${year} 年`
  if (month) text += `${month} 月`
  if (day) text += `${day} 日`
  return text
}

export function timeAgo(iso: string): string {
  if (!iso) return ''
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return ''
  const diff = Date.now() - then
  const minute = 60_000
  const hour = 60 * minute
  const day = 24 * hour
  if (diff < minute) return '刚刚'
  if (diff < hour) return `${Math.floor(diff / minute)} 分钟前`
  if (diff < day) return `${Math.floor(diff / hour)} 小时前`
  if (diff < 30 * day) return `${Math.floor(diff / day)} 天前`
  const d = new Date(iso)
  return `${d.getFullYear()}.${pad(d.getMonth() + 1)}.${pad(d.getDate())}`
}

/** 民国纪年，用于副标 */
export function republicYear(year: number): string {
  const n = year - 1911
  if (n <= 0) return '前民国'
  return `民国 ${n} 年`
}

const FOLK_TONES = ['#96703f', '#3f4c58', '#5c6b4f', '#7c241b', '#6b4a5a']

export function branchTone(official: boolean, id: number): string {
  if (official) return '#1b1814'
  return FOLK_TONES[Math.abs(id) % FOLK_TONES.length]
}

export function initialOf(name: string): string {
  return (name || '史').trim().slice(0, 1)
}

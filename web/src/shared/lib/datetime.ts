const dateTimeFormatter = new Intl.DateTimeFormat('zh-CN', {
  dateStyle: 'medium',
  timeStyle: 'short',
  timeZone: 'Asia/Shanghai',
})

const dateFormatter = new Intl.DateTimeFormat('zh-CN', {
  dateStyle: 'medium',
  timeZone: 'Asia/Shanghai',
})

/** 把服务端时间字符串格式化为上海时区的日期时间。 */
export function formatDateTime(raw: string): string {
  if (!raw) {
    return '—'
  }
  const date = new Date(raw)
  return Number.isNaN(date.getTime()) ? raw : dateTimeFormatter.format(date)
}

/** 把日期字符串格式化为上海时区的日期。 */
export function formatDate(raw: string): string {
  if (!raw) {
    return '—'
  }
  const date = new Date(raw)
  return Number.isNaN(date.getTime()) ? raw : dateFormatter.format(date)
}

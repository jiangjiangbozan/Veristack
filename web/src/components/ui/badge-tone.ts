/** 徽标色调。 */
export type Tone = 'danger' | 'info' | 'neutral' | 'success' | 'warning'

/** 根据状态文本映射到徽标色调。 */
export function toneForStatus(status: string): Tone {
  if (status === 'approved' || status === 'completed' || status === 'active') {
    return 'success'
  }
  if (status === 'pending' || status === 'in_progress') {
    return 'warning'
  }
  if (status === 'rejected' || status === 'archived') {
    return 'danger'
  }
  return 'neutral'
}

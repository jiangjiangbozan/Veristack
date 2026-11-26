import { request } from '@/services/http-client'
import type { VerificationResult } from '@/shared/types/acceptance'

/** 申请邮箱验证码，客户在提交验收结论前必须完成验证。 */
export function requestVerificationCode(token: string, email: string): Promise<{ sent: boolean }> {
  return request(`/share/${token}/verification-code`, {
    body: JSON.stringify({ email }),
    method: 'POST',
  })
}

/** 提交验证码完成邮箱验证。 */
export function verifyCustomerEmail(token: string, email: string, code: string): Promise<VerificationResult> {
  return request(`/share/${token}/verify`, {
    body: JSON.stringify({ code, email }),
    method: 'POST',
  })
}

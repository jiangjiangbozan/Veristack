/** 客户验收门户的邮箱验证请求。 */
export interface VerificationRequest {
  email: string
}

/** 客户邮箱验证结果。 */
export interface VerificationResult {
  verified: boolean
}

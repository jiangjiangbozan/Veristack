/** 用户角色。 */
export type UserRole = 'admin' | 'customer' | 'developer' | 'owner' | 'reviewer'

/** 用户视图。 */
export interface UserView {
  createdAt: string
  displayName: string
  email: string
  emailVerifiedAt: string
  id: string
  organizationId: string
  role: UserRole
  status: string
}

/** 项目视图。 */
export interface ProjectView {
  archived: boolean
  clientName: string
  contractNo: string
  createdAt: string
  description: string
  endDate: string
  id: string
  name: string
  startDate: string
  status: string
  updatedAt: string
}

/** 项目概览统计。 */
export interface ProjectOverview {
  baselineCount: number
  changeRequestCount: number
  milestoneCount: number
  requirementCount: number
}

/** 需求验收条件。 */
export interface RequirementView {
  acceptanceCriteria: string[]
  code: string
  description: string
  id: string
  priority: 'high' | 'low' | 'medium'
  status: string
  title: string
}

/** 范围基线视图。 */
export interface BaselineView {
  changeRequestId: string
  createdAt: string
  frozenAt: string
  id: string
  projectId: string
  requirements: RequirementView[]
  version: number
}

/** 两个基线之间的需求差异。 */
export interface BaselineDiff {
  added: RequirementView[]
  modified: RequirementView[]
  removed: RequirementView[]
}

/** 变更申请视图。 */
export interface ChangeRequestView {
  baselineId: string
  createdAt: string
  decidedAt: string
  id: string
  impact: string
  projectId: string
  reason: string
  requesterId: string
  status: 'approved' | 'pending' | 'rejected'
  type: string
}

/** 里程碑视图。 */
export interface MilestoneView {
  createdAt: string
  id: string
  name: string
  projectId: string
  requirementIds: string[]
  status: string
  targetDate: string
  updatedAt: string
}

/** 项目成员视图。 */
export interface MemberView {
  displayName: string
  email: string
  id: string
  role: string
}

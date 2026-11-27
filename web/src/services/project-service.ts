import { request } from '@/services/http-client'
import type { ApiPage } from '@/shared/types/api'
import type {
  BaselineDiff,
  BaselineView,
  ChangeRequestView,
  MemberView,
  MilestoneView,
  ProjectOverview,
  ProjectView,
} from '@/shared/types/domain'

interface PageResult<T> {
  items: T[]
  page?: ApiPage
}

/** 添加项目成员。 */
export function addMember(projectId: string, email: string, role: string): Promise<{ member: MemberView }> {
  return request(`/projects/${projectId}/members`, {
    body: JSON.stringify({ email, role }),
    method: 'POST',
  })
}

/** 创建范围基线。 */
export function createBaseline(
  projectId: string,
  requirements: Array<{
    acceptanceCriteria: string[]
    code: string
    description: string
    priority: string
    title: string
  }>,
): Promise<{ baseline: BaselineView }> {
  return request(`/projects/${projectId}/scope-baselines`, {
    body: JSON.stringify({ requirements }),
    method: 'POST',
  })
}

/** 发起变更申请。 */
export function createChangeRequest(
  projectId: string,
  payload: {
    impact: string
    reason: string
    requirements: Array<{
      acceptanceCriteria: string[]
      code: string
      description: string
      operation: string
      priority: string
      title: string
    }>
    type: string
  },
): Promise<{ changeRequest: ChangeRequestView }> {
  return request(`/projects/${projectId}/change-requests`, {
    body: JSON.stringify(payload),
    method: 'POST',
  })
}

/** 创建里程碑。 */
export function createMilestone(
  projectId: string,
  payload: { name: string; requirementIds: string[]; targetDate: string },
): Promise<{ milestone: MilestoneView }> {
  return request(`/projects/${projectId}/milestones`, {
    body: JSON.stringify(payload),
    method: 'POST',
  })
}

/** 导入项目的需求输入。 */
export interface ImportRequirementInput {
  acceptanceCriteria: string[] // 验收条件
  code: string // 需求编号
  description: string // 需求描述
  priority: string // 优先级
  title: string // 需求标题
}

/** 导入项目的里程碑输入。 */
export interface ImportMilestoneInput {
  name: string // 里程碑名称
  requirementCodes: string[] // 关联需求编号
  targetDate: string // 目标日期
}

/** 导入已有项目的请求体。 */
export interface ImportProjectPayload {
  milestones: ImportMilestoneInput[] // 里程碑列表
  project: {
    clientName: string // 客户名称
    contractNo: string // 合同编号
    description: string // 项目描述
    endDate: string // 结束日期
    name: string // 项目名称
    startDate: string // 开始日期
  }
  requirements: ImportRequirementInput[] // 初始需求列表
}

/** 导入已有项目，一次请求内创建项目、v1 基线与里程碑。 */
export function importProject(payload: ImportProjectPayload): Promise<{
  milestoneCount: number
  project: ProjectView
  requirementCount: number
}> {
  return request('/projects/import', { body: JSON.stringify(payload), method: 'POST' })
}

/** 创建项目。 */
export function createProject(payload: {
  clientName: string
  contractNo: string
  description: string
  endDate: string
  name: string
  startDate: string
}): Promise<{ project: ProjectView }> {
  return request('/projects', { body: JSON.stringify(payload), method: 'POST' })
}

/** 对变更申请做出决定。 */
export function decideChangeRequest(
  changeId: string,
  decision: string,
  comment: string,
): Promise<{ baseline: BaselineView | null; changeRequest: ChangeRequestView }> {
  return request(`/change-requests/${changeId}/decision`, {
    body: JSON.stringify({ comment, decision }),
    method: 'POST',
  })
}

/** 对比两个需求基线。 */
export function fetchBaselineDiff(baselineId: string, againstId: string): Promise<BaselineDiff> {
  return request(`/scope-baselines/${baselineId}/diff?against=${againstId}`)
}

/** 删除里程碑。 */
export function deleteMilestone(milestoneId: string): Promise<{ deleted: boolean }> {
  return request(`/milestones/${milestoneId}`, { method: 'DELETE' })
}

/** 删除项目。 */
export function deleteProject(projectId: string): Promise<{ deleted: boolean }> {
  return request(`/projects/${projectId}`, { method: 'DELETE' })
}

/** 读取项目详情。 */
export function fetchProject(projectId: string): Promise<{ project: ProjectView }> {
  return request(`/projects/${projectId}`)
}

/** 读取项目概览统计。 */
export function fetchOverview(projectId: string): Promise<ProjectOverview> {
  return request(`/projects/${projectId}/overview`)
}

/** 列出范围基线。 */
export function listBaselines(projectId: string): Promise<{ items: BaselineView[] }> {
  return request(`/projects/${projectId}/scope-baselines`)
}

/** 列出变更申请。 */
export function listChangeRequests(projectId: string, page = 1, pageSize = 20): Promise<PageResult<ChangeRequestView>> {
  return request(`/projects/${projectId}/change-requests?page=${page}&pageSize=${pageSize}`)
}

/** 列出项目成员。 */
export function listMembers(projectId: string): Promise<{ items: MemberView[] }> {
  return request(`/projects/${projectId}/members`)
}

/** 列出里程碑。 */
export function listMilestones(projectId: string): Promise<{ items: MilestoneView[] }> {
  return request(`/projects/${projectId}/milestones`)
}

/** 分页列出项目。 */
export function listProjects(page = 1, pageSize = 20): Promise<PageResult<ProjectView>> {
  return request(`/projects?page=${page}&pageSize=${pageSize}`)
}

/** 移除项目成员。 */
export function removeMember(projectId: string, userId: string): Promise<{ deleted: boolean }> {
  return request(`/projects/${projectId}/members/${userId}`, { method: 'DELETE' })
}

/** 更新里程碑。 */
export function updateMilestone(
  milestoneId: string,
  payload: { name: string; requirementIds: string[]; status: string; targetDate: string },
): Promise<{ milestone: MilestoneView }> {
  return request(`/milestones/${milestoneId}`, {
    body: JSON.stringify(payload),
    method: 'PUT',
  })
}

/** 更新项目。 */
export function updateProject(
  projectId: string,
  payload: {
    archived: boolean
    clientName: string
    contractNo: string
    description: string
    endDate: string
    name: string
    startDate: string
    status: string
  },
): Promise<{ project: ProjectView }> {
  return request(`/projects/${projectId}`, { body: JSON.stringify(payload), method: 'PUT' })
}

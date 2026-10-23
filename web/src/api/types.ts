export interface UserBrief {
  id: number
  username: string
  nickname: string
}

export interface BranchCard {
  id: number
  name: string
  description: string
  official: boolean
  owner: UserBrief | null
  parentId: number
  parentName: string
  forkEventId: number
  forkEventTitle: string
  forkYear: number
  inheritedCount: number
  ownCount: number
  eventCount: number
  forkCount: number
  likeCount: number
  commentCount: number
  firstYear: number
  lastYear: number
  liked: boolean
  canWrite: boolean
  createdAt: string
}

export interface LineageItem {
  id: number
  name: string
  official: boolean
  forkYear: number
  eventCount: number
}

export interface BranchDetail extends BranchCard {
  ancestors: LineageItem[]
  children: BranchCard[]
  siblings: BranchCard[]
  isOwner: boolean
}

export interface TimelineItem {
  id: number
  branchId: number
  branchName: string
  branchOfficial: boolean
  title: string
  content?: string
  excerpt: string
  year: number
  month: number
  day: number
  location: string
  figures: string[]
  tags: string[]
  author: UserBrief | null
  likeCount: number
  commentCount: number
  liked: boolean
  inherited: boolean
  createdAt: string
}

export interface EventDetail extends TimelineItem {
  branch: BranchCard
  canEdit: boolean
}

export interface CommentView {
  id: number
  targetType: string
  targetId: number
  replyToId: number
  replyToName: string
  content: string
  user: UserBrief | null
  likeCount: number
  liked: boolean
  canDelete: boolean
  createdAt: string
}

export interface GraphNode {
  id: number
  name: string
  official: boolean
  parentId: number
  forkEventId: number
  forkYear: number
  startYear: number
  endYear: number
  eventCount: number
  forkCount: number
  likeCount: number
  owner: string
}

export interface Stats {
  officialEvents: number
  folkEvents: number
  totalEvents: number
  totalBranches: number
  officialBranches: number
  folkBranches: number
  chroniclers: number
  totalLikes: number
  todayEvents: number
}

export interface Page<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

export interface LikeResult {
  liked: boolean
  count: number
}

export interface AuthResult {
  token: string
  user: UserBrief
}

export interface MeView {
  id: number
  username: string
  nickname: string
  bio: string
  createdAt: string
  branchCount: number
  eventCount: number
}

export interface EventPayload {
  title: string
  content: string
  year: number
  month: number
  day: number
  location: string
  figures: string[]
  tags: string[]
}

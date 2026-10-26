import { request } from './http'
import type {
  AuthResult,
  BranchCard,
  BranchDetail,
  CommentView,
  EventDetail,
  EventPayload,
  GraphNode,
  LikeResult,
  MeView,
  Page,
  Stats,
  TimelineItem,
} from './types'

export const api = {
  stats: () => request.get<Stats>('/stats'),
  graph: () => request.get<GraphNode[]>('/graph'),

  branches: (params: { scope?: string; sort?: string; q?: string; page?: number; pageSize?: number } = {}) =>
    request.get<Page<BranchCard>>('/branches', params),
  branch: (id: number) => request.get<BranchDetail>(`/branches/${id}`),
  branchTimeline: (id: number, page = 1, pageSize = 20, own = false, desc = false) =>
    request.get<Page<TimelineItem>>(`/branches/${id}/timeline`, {
      page,
      pageSize,
      ...(own ? { own: 1 } : {}),
      ...(desc ? { order: 'desc' } : {}),
    }),
  fork: (payload: { parentId: number; forkEventId: number; name: string; description: string }) =>
    request.post<BranchDetail>('/branches', payload),
  updateBranch: (id: number, payload: { name: string; description: string }) =>
    request.patch<BranchDetail>(`/branches/${id}`, payload),
  deleteBranch: (id: number) => request.delete<{ deleted: number }>(`/branches/${id}`),
  myBranches: (page = 1, pageSize = 20) =>
    request.get<Page<BranchCard>>('/me/branches', { page, pageSize }),

  events: (params: { scope?: string; sort?: string; q?: string; page?: number; pageSize?: number } = {}) =>
    request.get<Page<TimelineItem>>('/events', params),
  event: (id: number) => request.get<EventDetail>(`/events/${id}`),
  createEvent: (branchId: number, payload: EventPayload) =>
    request.post<EventDetail>(`/branches/${branchId}/events`, payload),
  updateEvent: (id: number, payload: EventPayload) => request.patch<EventDetail>(`/events/${id}`, payload),
  deleteEvent: (id: number) => request.delete<{ deleted: number }>(`/events/${id}`),
  myEvents: (page = 1, pageSize = 20) => request.get<Page<TimelineItem>>('/me/events', { page, pageSize }),

  toggleLike: (targetType: string, targetId: number) =>
    request.post<LikeResult>('/likes', { targetType, targetId }),
  comments: (targetType: string, targetId: number, page = 1, pageSize = 50) =>
    request.get<Page<CommentView>>('/comments', { targetType, targetId, page, pageSize }),
  createComment: (payload: { targetType: string; targetId: number; content: string; replyToId?: number }) =>
    request.post<CommentView>('/comments', payload),
  deleteComment: (id: number) => request.delete<{ deleted: number }>(`/comments/${id}`),

  login: (username: string, password: string) =>
    request.post<AuthResult>('/auth/login', { username, password }),
  register: (username: string, password: string, nickname: string) =>
    request.post<AuthResult>('/auth/register', { username, password, nickname }),
  me: () => request.get<MeView>('/auth/me'),
}

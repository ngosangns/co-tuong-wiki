import type {
  CombinedLineMoveWindow,
  CombinedNextStepsRequest,
  CombinedStepMoveWindow,
  Lesson,
  LessonPhase,
  LessonSummary,
} from './types'
import type { LessonMove, Side } from '../core/xiangqi'
import type { EngineEvaluation } from '../engine/types'

const apiBaseUrl = import.meta.env.VITE_API_URL ?? 'http://127.0.0.1:8090'

function messageFromErrorBody(body: string, status: number) {
  if (!body) return `Request failed with ${status}`

  try {
    const payload = JSON.parse(body) as unknown

    if (payload && typeof payload === 'object') {
      const record = payload as Record<string, unknown>
      if (typeof record.error === 'string' && record.error) return record.error
      if (typeof record.message === 'string' && record.message) return record.message
    }
  } catch {
    // Non-JSON error bodies can be shown as-is.
  }

  return body
}

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  })

  if (!response.ok) {
    const body = await response.text()
    throw new Error(messageFromErrorBody(body, response.status))
  }

  return response.json() as Promise<T>
}

export function fetchCategories() {
  return fetchJSON<string[]>('/api/categories')
}

export function fetchLessons(category?: string) {
  const params = new URLSearchParams()
  if (category) params.set('category', category)
  const suffix = params.toString() ? `?${params.toString()}` : ''
  return fetchJSON<LessonSummary[]>(`/api/lessons${suffix}`)
}

export function fetchLesson(id: string) {
  return fetchJSON<Lesson>(`/api/lessons/${encodeURIComponent(id)}`)
}

export function fetchCombinedLesson() {
  return fetchJSON<Lesson>('/api/combined-lesson')
}

export function fetchCombinedLineMoves(lineId: string, from: number, limit = 12) {
  const params = new URLSearchParams({
    from: String(Math.max(0, from)),
    limit: String(limit),
  })
  return fetchJSON<CombinedLineMoveWindow>(`/api/combined-lesson/lines/${encodeURIComponent(lineId)}/moves?${params.toString()}`)
}

export function fetchCombinedStepMoves(from: number, limit = 1, phase?: LessonPhase) {
  const params = new URLSearchParams({
    from: String(Math.max(0, from)),
    limit: String(limit),
  })
  if (phase) params.set('phase', phase)
  return fetchJSON<CombinedStepMoveWindow>(`/api/combined-lesson/moves?${params.toString()}`)
}

export function fetchCombinedNextSteps(request: CombinedNextStepsRequest) {
  return fetchJSON<CombinedStepMoveWindow>('/api/combined-lesson/next-steps', {
    method: 'POST',
    body: JSON.stringify({
      ...request,
      from: Math.max(0, request.from),
      limit: request.limit ?? 1,
    }),
  })
}

export function analyzePosition(
  input: {
    fen: string
    sideToMove: Side
    nextMove?: LessonMove
  },
  signal?: AbortSignal,
) {
  return fetchJSON<EngineEvaluation>('/api/analyze', {
    method: 'POST',
    signal,
    body: JSON.stringify(input),
  })
}

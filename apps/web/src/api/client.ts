import type {
  CombinedLineMoveWindow,
  CombinedNextStepsRequest,
  CombinedStepMoveWindow,
  Lesson,
  LessonPhase,
  LessonSummary,
  LineEvaluationRequest,
  LineEvaluationResponse,
  OpeningBookResponse,
} from './types'
import type { LessonMove, Side } from '../core/xiangqi'
import type { EngineEvaluation } from '../engine/types'
import { apiCacheKey, cachedJSON } from './cache'

const apiBaseUrl = import.meta.env.VITE_API_URL ?? 'http://127.0.0.1:8090'
const catalogCacheTTL = 5 * 60 * 1000
const combinedCacheTTL = 10 * 60 * 1000
const analysisCacheTTL = 10 * 60 * 1000

interface CacheOptions {
  key?: string
  ttlMs?: number
  persist?: boolean
}

async function fetchJSON<T>(path: string, init?: RequestInit, cache?: CacheOptions): Promise<T> {
  return cachedJSON<T>(`${apiBaseUrl}${path}`, init, cache)
}

export function fetchCategories() {
  const path = '/api/categories'
  return fetchJSON<string[]>(path, undefined, {
    key: apiCacheKey('GET', path),
    ttlMs: catalogCacheTTL,
    persist: true,
  })
}

export function fetchLessons(category?: string) {
  const params = new URLSearchParams()
  if (category) params.set('category', category)
  const suffix = params.toString() ? `?${params.toString()}` : ''
  const path = `/api/lessons${suffix}`
  return fetchJSON<LessonSummary[]>(path, undefined, {
    key: apiCacheKey('GET', path),
    ttlMs: catalogCacheTTL,
    persist: true,
  })
}

export function fetchLesson(id: string) {
  const path = `/api/lessons/${encodeURIComponent(id)}`
  return fetchJSON<Lesson>(path, undefined, {
    key: apiCacheKey('GET', path),
    ttlMs: catalogCacheTTL,
    persist: true,
  })
}

export function fetchCombinedLesson() {
  const path = '/api/combined-lesson'
  return fetchJSON<Lesson>(path, undefined, {
    key: apiCacheKey('GET', path),
    ttlMs: combinedCacheTTL,
    persist: true,
  })
}

export function fetchCombinedLineMoves(lineId: string, from: number, limit = 12) {
  const params = new URLSearchParams({
    from: String(Math.max(0, from)),
    limit: String(limit),
  })
  const path = `/api/combined-lesson/lines/${encodeURIComponent(lineId)}/moves?${params.toString()}`
  return fetchJSON<CombinedLineMoveWindow>(path, undefined, {
    key: apiCacheKey('GET', path),
    ttlMs: combinedCacheTTL,
  })
}

export function fetchCombinedStepMoves(from: number, limit = 1, phase?: LessonPhase) {
  const params = new URLSearchParams({
    from: String(Math.max(0, from)),
    limit: String(limit),
  })
  if (phase) params.set('phase', phase)
  const path = `/api/combined-lesson/moves?${params.toString()}`
  return fetchJSON<CombinedStepMoveWindow>(path, undefined, {
    key: apiCacheKey('GET', path),
    ttlMs: combinedCacheTTL,
  })
}

export function fetchCombinedNextSteps(request: CombinedNextStepsRequest) {
  const path = '/api/combined-lesson/next-steps'
  const body = {
    ...request,
    from: Math.max(0, request.from),
    limit: request.limit ?? 1,
  }
  return fetchJSON<CombinedStepMoveWindow>(
    path,
    {
      method: 'POST',
      body: JSON.stringify(body),
    },
    {
      key: apiCacheKey('POST', path, body),
      ttlMs: combinedCacheTTL,
    },
  )
}

export function analyzePosition(
  input: {
    fen: string
    sideToMove: Side
    nextMove?: LessonMove
  },
  signal?: AbortSignal,
) {
  const path = '/api/analyze'
  return fetchJSON<EngineEvaluation>(
    path,
    {
      method: 'POST',
      signal,
      body: JSON.stringify(input),
    },
    {
      key: apiCacheKey('POST', path, input),
      ttlMs: analysisCacheTTL,
    },
  )
}

export function evaluateLine(request: LineEvaluationRequest) {
  const path = '/api/line-evaluation'
  return fetchJSON<LineEvaluationResponse>(
    path,
    {
      method: 'POST',
      body: JSON.stringify(request),
    },
    {
      key: apiCacheKey('POST', path, request),
      ttlMs: 60 * 60 * 1000,
    },
  )
}

export function fetchOpeningBook(fen: string) {
  const path = `/api/opening?fen=${encodeURIComponent(fen)}`
  return fetchJSON<OpeningBookResponse>(path, undefined, {
    key: apiCacheKey('GET', path),
    ttlMs: 24 * 60 * 60 * 1000,
  })
}

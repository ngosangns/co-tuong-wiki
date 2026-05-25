import type { Lesson, LessonSummary } from './types'
import type { LessonMove, Side } from '../core/xiangqi'
import type { EngineEvaluation } from '../engine/types'

const apiBaseUrl = import.meta.env.VITE_API_URL ?? 'http://127.0.0.1:8090'

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  })

  if (!response.ok) {
    const message = await response.text()
    throw new Error(message || `Request failed with ${response.status}`)
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

import type { LessonMove } from '../core/xiangqi'
import type { EngineEvaluation } from '../engine/types'

export interface LessonChoice {
  id: string
  prompt: string
  options: Array<{
    moveId: string
    label: string
    verdict: 'correct' | 'danger'
    feedback: string
  }>
}

export interface LessonLine {
  id: string
  title: string
  description: string
  moves: LessonMove[]
}

export interface LessonSummary {
  id: string
  slug: string
  title: string
  summary: string
  category: string
  difficulty: string
  tags: string[]
}

export interface Lesson extends LessonSummary {
  principles: string[]
  initialFen?: string
  lines: LessonLine[]
  choice: LessonChoice
}

export type AnalyzeResponse = EngineEvaluation

import type { LessonMove } from '../core/xiangqi'
import type { EngineEvaluation } from '../engine/types'

export interface LessonChoice {
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
  moves: LessonMove[]
}

export interface LessonSummary {
  id: string
  title: string
  category: string
  difficulty: string
}

export interface Lesson extends LessonSummary {
  initialFen?: string
  lines: LessonLine[]
  choice: LessonChoice
}

export type AnalyzeResponse = EngineEvaluation

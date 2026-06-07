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
  initialFen?: string
  phase?: LessonPhase
  pieceCount?: number
  moveCount?: number
  moves?: LessonMove[]
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

export interface CombinedLineMoveWindow {
  lineId: string
  phase?: LessonPhase
  pieceCount?: number
  from: number
  moves: LessonMove[]
  totalMoves: number
}

export interface CombinedStepMoveWindow {
  from: number
  limit: number
  lines: CombinedLineMoveWindow[]
}

export interface CombinedNextStepsRequest {
  phase: LessonPhase
  initialFen: string
  from: number
  limit?: number
  prefix: LessonMove[]
}

export type AnalyzeResponse = EngineEvaluation

export type LessonPhase = 'opening' | 'middlegame' | 'endgame'

export interface LineEvaluationPly {
  ply: number
  fen: string
  sideToMove: 'red' | 'black'
  nextMove?: LessonMove
}

export interface LineEvaluationRequest {
  openingFen?: string
  engine?: 'pikafish' | 'fairy-stockfish'
  plies: LineEvaluationPly[]
}

export type MoveClassification =
  | 'best'
  | 'excellent'
  | 'good'
  | 'inaccuracy'
  | 'mistake'
  | 'blunder'
  | 'unknown'

export interface LineEvaluationPlyResult {
  ply: number
  fen: string
  sideToMove: 'red' | 'black'
  cp: number
  perspective: 'red' | 'black'
  bestMove?: string
  bestScore?: number
  pv: string[]
  depth: number
  status: 'ok' | 'error' | 'unavailable'
  error?: string
  cpLoss?: number
  classification?: MoveClassification
}

export interface LineEvaluationResponse {
  engine: 'pikafish' | 'fairy-stockfish'
  plies: LineEvaluationPlyResult[]
}

export interface OpeningBookMove {
  move: string
  notation: string
  name: string
  frequency: number
  popularity: number
}

export interface OpeningBookResponse {
  available: boolean
  fen: string
  visits: number
  moves: OpeningBookMove[]
  stats?: Record<string, unknown>
}

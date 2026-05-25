import type { BoardState, Coordinate, LessonMove, Side } from '../core/xiangqi'

export type EngineSource = 'uci' | 'wukong'

export type EngineStatus = 'idle' | 'analyzing' | 'ready' | 'error'

export interface EngineMove {
  from: Coordinate
  to: Coordinate
  notation: string
  score?: number
}

export interface EngineAnalyzeInput {
  board: BoardState
  fen: string
  sideToMove: Side
  nextMove?: LessonMove
  currentMove?: LessonMove
  moveNumber: number
}

export interface EngineEvaluation {
  fen: string
  sideToMove: Side
  score: {
    cp: number
    perspective: Side
  }
  bestMove?: EngineMove
  principalVariation: EngineMove[]
  depth: number
  source: EngineSource
  status: EngineStatus
  message?: string
}

export interface EngineAdapter {
  id: EngineSource
  label: string
  analyze(input: EngineAnalyzeInput, signal?: AbortSignal): Promise<EngineEvaluation>
  dispose?(): void
}

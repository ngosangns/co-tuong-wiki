import { describe, expect, it } from 'vitest'
import {
  applyMove,
  initialBoard,
  pieceAt,
  replayMoves,
  sameSquare,
} from './xiangqi'
import type { LessonMove } from './xiangqi'

const makeMove = (overrides: Partial<LessonMove> = {}): LessonMove => ({
  id: 'm1',
  side: 'red',
  from: { file: 4, rank: 9 },
  to: { file: 4, rank: 8 },
  comment: '',
  ...overrides,
})

describe('xiangqi core', () => {
  it('initialBoard has 32 pieces in standard layout', () => {
    expect(initialBoard).toHaveLength(32)
    const reds = initialBoard.filter((p) => p.side === 'red')
    const blacks = initialBoard.filter((p) => p.side === 'black')
    expect(reds).toHaveLength(16)
    expect(blacks).toHaveLength(16)
  })

  it('sameSquare compares file and rank', () => {
    expect(sameSquare({ file: 0, rank: 0 }, { file: 0, rank: 0 })).toBe(true)
    expect(sameSquare({ file: 0, rank: 0 }, { file: 1, rank: 0 })).toBe(false)
    expect(sameSquare({ file: 0, rank: 0 }, { file: 0, rank: 1 })).toBe(false)
  })

  it('pieceAt returns the piece at coordinate or undefined', () => {
    const general = pieceAt(initialBoard, { file: 4, rank: 9 })
    expect(general?.kind).toBe('general')
    expect(general?.side).toBe('red')
    expect(pieceAt(initialBoard, { file: 0, rank: 4 })).toBeUndefined()
  })

  it('applyMove moves a piece and keeps board size when no capture', () => {
    const after = applyMove(initialBoard, makeMove())
    expect(after).toHaveLength(32)
    const moved = pieceAt(after, { file: 4, rank: 8 })
    expect(moved?.kind).toBe('general')
    expect(pieceAt(after, { file: 4, rank: 9 })).toBeUndefined()
  })

  it('applyMove captures a target piece', () => {
    const board = [
      { id: 'a', side: 'red' as const, kind: 'chariot' as const, label: '车', position: { file: 0, rank: 5 } },
      { id: 'b', side: 'black' as const, kind: 'soldier' as const, label: '卒', position: { file: 0, rank: 4 } },
    ]
    const after = applyMove(board, makeMove({ from: { file: 0, rank: 5 }, to: { file: 0, rank: 4 } }))
    expect(after).toHaveLength(1)
    expect(after[0].id).toBe('a')
    expect(after[0].position).toEqual({ file: 0, rank: 4 })
  })

  it('applyMove throws when the source square is empty', () => {
    expect(() => applyMove(initialBoard, makeMove({ from: { file: 0, rank: 4 } }))).toThrow(/Cannot replay move/)
  })

  it('replayMoves accumulates state across a sequence', () => {
    const moves: LessonMove[] = [
      makeMove({ id: 'm1', from: { file: 4, rank: 9 }, to: { file: 4, rank: 8 } }),
      makeMove({ id: 'm2', side: 'black', from: { file: 4, rank: 0 }, to: { file: 4, rank: 1 } }),
    ]
    const final = replayMoves(initialBoard, moves)
    expect(pieceAt(final, { file: 4, rank: 8 })?.side).toBe('red')
    expect(pieceAt(final, { file: 4, rank: 1 })?.side).toBe('black')
  })
})

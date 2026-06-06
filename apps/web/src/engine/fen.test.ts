import { describe, expect, it } from 'vitest'
import { boardFromXiangqiFen, boardToXiangqiFen, engineNotationToMove } from './fen'
import { initialBoard, pieceAt } from '../core/xiangqi'
import { engineSquareLabel } from '../composables/useMoveTrie'

describe('Xiangqi FEN', () => {
  it('parses the standard starting FEN into 32 pieces', () => {
    const board = boardFromXiangqiFen('rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1')
    expect(board).toHaveLength(32)
    expect(pieceAt(board, { file: 4, rank: 0 })?.kind).toBe('general')
    expect(pieceAt(board, { file: 4, rank: 0 })?.side).toBe('black')
    expect(pieceAt(board, { file: 4, rank: 9 })?.kind).toBe('general')
    expect(pieceAt(board, { file: 4, rank: 9 })?.side).toBe('red')
  })

  it('parses an empty 9/9/9/.../9 board with 0 pieces', () => {
    const board = boardFromXiangqiFen('9/9/9/9/9/9/9/9/9/9 w - - 0 1')
    expect(board).toHaveLength(0)
  })

  it('rejects FEN with wrong number of ranks', () => {
    expect(() => boardFromXiangqiFen('rnbakabnr/9 w - - 0 1')).toThrow(/10 ranks/)
  })

  it('rejects FEN with rank not summing to 9 files', () => {
    expect(() => boardFromXiangqiFen('8/9/9/9/9/9/9/9/9/9 w - - 0 1')).toThrow(/8 files/)
  })

  it('rejects unsupported piece symbol', () => {
    expect(() => boardFromXiangqiFen('xnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1')).toThrow(/Unsupported/)
  })

  it('round-trips initial board through FEN', () => {
    const fen = boardToXiangqiFen(initialBoard, 'red', 1)
    expect(fen).toBe('rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1')
    const parsed = boardFromXiangqiFen(fen)
    expect(parsed).toHaveLength(32)
  })

  it('engineSquareLabel maps (file, rank) to algebraic square', () => {
    // Board rank 0 (top, black) -> engine rank 9
    expect(engineSquareLabel(0, 0)).toBe('a9')
    // Board rank 9 (bottom, red) -> engine rank 0
    expect(engineSquareLabel(4, 9)).toBe('e0')
    expect(engineSquareLabel(8, 5)).toBe('i4')
  })

  it('engineNotationToMove parses a 4-char UCI string', () => {
    const parsed = engineNotationToMove('b2e2')
    expect(parsed?.from).toEqual({ file: 1, rank: 7 })
    expect(parsed?.to).toEqual({ file: 4, rank: 7 })
    expect(parsed?.notation).toBe('b2e2')
  })

  it('engineNotationToMove returns null for malformed input', () => {
    expect(engineNotationToMove('z9a0')).toBeNull()
    expect(engineNotationToMove('a')).toBeNull()
  })
})

export type Side = 'red' | 'black'

export type PieceKind = 'general' | 'advisor' | 'elephant' | 'horse' | 'chariot' | 'cannon' | 'soldier'

export interface Coordinate {
  file: number
  rank: number
}

export interface Piece {
  id: string
  side: Side
  kind: PieceKind
  label: string
  position: Coordinate
}

export interface LessonMove {
  id: string
  side: Side
  from: Coordinate
  to: Coordinate
  comment: string
}

export type BoardState = Piece[]

export const initialBoard: BoardState = [
  { id: 'br1', side: 'black', kind: 'chariot', label: '车', position: { file: 0, rank: 0 } },
  { id: 'bh1', side: 'black', kind: 'horse', label: '马', position: { file: 1, rank: 0 } },
  { id: 'be1', side: 'black', kind: 'elephant', label: '象', position: { file: 2, rank: 0 } },
  { id: 'ba1', side: 'black', kind: 'advisor', label: '士', position: { file: 3, rank: 0 } },
  { id: 'bg', side: 'black', kind: 'general', label: '将', position: { file: 4, rank: 0 } },
  { id: 'ba2', side: 'black', kind: 'advisor', label: '士', position: { file: 5, rank: 0 } },
  { id: 'be2', side: 'black', kind: 'elephant', label: '象', position: { file: 6, rank: 0 } },
  { id: 'bh2', side: 'black', kind: 'horse', label: '马', position: { file: 7, rank: 0 } },
  { id: 'br2', side: 'black', kind: 'chariot', label: '车', position: { file: 8, rank: 0 } },
  { id: 'bc1', side: 'black', kind: 'cannon', label: '炮', position: { file: 1, rank: 2 } },
  { id: 'bc2', side: 'black', kind: 'cannon', label: '炮', position: { file: 7, rank: 2 } },
  { id: 'bs1', side: 'black', kind: 'soldier', label: '卒', position: { file: 0, rank: 3 } },
  { id: 'bs2', side: 'black', kind: 'soldier', label: '卒', position: { file: 2, rank: 3 } },
  { id: 'bs3', side: 'black', kind: 'soldier', label: '卒', position: { file: 4, rank: 3 } },
  { id: 'bs4', side: 'black', kind: 'soldier', label: '卒', position: { file: 6, rank: 3 } },
  { id: 'bs5', side: 'black', kind: 'soldier', label: '卒', position: { file: 8, rank: 3 } },
  { id: 'rs1', side: 'red', kind: 'soldier', label: '兵', position: { file: 0, rank: 6 } },
  { id: 'rs2', side: 'red', kind: 'soldier', label: '兵', position: { file: 2, rank: 6 } },
  { id: 'rs3', side: 'red', kind: 'soldier', label: '兵', position: { file: 4, rank: 6 } },
  { id: 'rs4', side: 'red', kind: 'soldier', label: '兵', position: { file: 6, rank: 6 } },
  { id: 'rs5', side: 'red', kind: 'soldier', label: '兵', position: { file: 8, rank: 6 } },
  { id: 'rc1', side: 'red', kind: 'cannon', label: '炮', position: { file: 1, rank: 7 } },
  { id: 'rc2', side: 'red', kind: 'cannon', label: '炮', position: { file: 7, rank: 7 } },
  { id: 'rr1', side: 'red', kind: 'chariot', label: '车', position: { file: 0, rank: 9 } },
  { id: 'rh1', side: 'red', kind: 'horse', label: '马', position: { file: 1, rank: 9 } },
  { id: 're1', side: 'red', kind: 'elephant', label: '相', position: { file: 2, rank: 9 } },
  { id: 'ra1', side: 'red', kind: 'advisor', label: '仕', position: { file: 3, rank: 9 } },
  { id: 'rg', side: 'red', kind: 'general', label: '帅', position: { file: 4, rank: 9 } },
  { id: 'ra2', side: 'red', kind: 'advisor', label: '仕', position: { file: 5, rank: 9 } },
  { id: 're2', side: 'red', kind: 'elephant', label: '相', position: { file: 6, rank: 9 } },
  { id: 'rh2', side: 'red', kind: 'horse', label: '马', position: { file: 7, rank: 9 } },
  { id: 'rr2', side: 'red', kind: 'chariot', label: '车', position: { file: 8, rank: 9 } },
]

export function sameSquare(a: Coordinate, b: Coordinate) {
  return a.file === b.file && a.rank === b.rank
}

export function applyMove(board: BoardState, move: LessonMove): BoardState {
  const movingPiece = board.find((piece) => sameSquare(piece.position, move.from) && piece.side === move.side)

  if (!movingPiece) {
    throw new Error(`Cannot replay move from (${move.from.file},${move.from.rank}) to (${move.to.file},${move.to.rank}).`)
  }

  // Replay data is curated lesson content, so this only applies captures and movement.
  return board
    .filter((piece) => piece.id === movingPiece.id || !sameSquare(piece.position, move.to))
    .map((piece) => (piece.id === movingPiece.id ? { ...piece, position: { ...move.to } } : { ...piece }))
}

export function replayMoves(board: BoardState, moves: LessonMove[]) {
  return moves.reduce((state, move) => applyMove(state, move), board)
}

export function pieceAt(board: BoardState, coordinate: Coordinate) {
  return board.find((piece) => sameSquare(piece.position, coordinate))
}

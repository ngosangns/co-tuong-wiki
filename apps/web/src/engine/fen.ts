import type { BoardState, Coordinate, PieceKind, Side } from '../core/xiangqi'

const files = 'abcdefghi'

const pieceFen: Record<PieceKind, string> = {
  general: 'k',
  advisor: 'a',
  elephant: 'b',
  horse: 'n',
  chariot: 'r',
  cannon: 'c',
  soldier: 'p',
}

const fenPieceKind: Record<string, PieceKind> = {
  k: 'general',
  a: 'advisor',
  b: 'elephant',
  n: 'horse',
  h: 'horse',
  r: 'chariot',
  c: 'cannon',
  p: 'soldier',
}

const pieceLabels: Record<Side, Record<PieceKind, string>> = {
  red: {
    general: '帅',
    advisor: '仕',
    elephant: '相',
    horse: '马',
    chariot: '车',
    cannon: '炮',
    soldier: '兵',
  },
  black: {
    general: '将',
    advisor: '士',
    elephant: '象',
    horse: '马',
    chariot: '车',
    cannon: '炮',
    soldier: '卒',
  },
}

export function sideToFenTurn(side: Side) {
  return side === 'red' ? 'w' : 'b'
}

export function boardToXiangqiFen(board: BoardState, sideToMove: Side, moveNumber = 1) {
  const placement = Array.from({ length: 10 }, (_, rank) => {
    let row = ''
    let empty = 0

    for (let file = 0; file < 9; file += 1) {
      const piece = board.find((item) => item.position.file === file && item.position.rank === rank)

      if (!piece) {
        empty += 1
        continue
      }

      if (empty > 0) {
        row += empty.toString()
        empty = 0
      }

      const symbol = pieceFen[piece.kind]
      row += piece.side === 'red' ? symbol.toUpperCase() : symbol
    }

    return row + (empty > 0 ? empty.toString() : '')
  }).join('/')

  return `${placement} ${sideToFenTurn(sideToMove)} - - 0 ${moveNumber}`
}

export function boardFromXiangqiFen(fen: string): BoardState {
  const placement = fen.trim().split(/\s+/)[0]
  const ranks = placement.split('/')

  if (ranks.length !== 10) {
    throw new Error('A Xiangqi FEN placement must contain 10 ranks.')
  }

  const counters: Record<Side, Record<PieceKind, number>> = {
    red: { general: 0, advisor: 0, elephant: 0, horse: 0, chariot: 0, cannon: 0, soldier: 0 },
    black: { general: 0, advisor: 0, elephant: 0, horse: 0, chariot: 0, cannon: 0, soldier: 0 },
  }

  return ranks.flatMap((rankPlacement, rank) => {
    const pieces: BoardState = []
    let file = 0

    for (const symbol of rankPlacement) {
      if (/\d/.test(symbol)) {
        file += Number(symbol)
        continue
      }

      const side: Side = symbol === symbol.toUpperCase() ? 'red' : 'black'
      const kind = fenPieceKind[symbol.toLowerCase()]

      if (!kind) {
        throw new Error(`Unsupported Xiangqi FEN piece "${symbol}".`)
      }

      counters[side][kind] += 1
      pieces.push({
        id: `${side[0]}-${kind}-${counters[side][kind]}`,
        side,
        kind,
        label: pieceLabels[side][kind],
        position: { file, rank },
      })
      file += 1
    }

    if (file !== 9) {
      throw new Error(`Xiangqi FEN rank ${rank + 1} has ${file} files instead of 9.`)
    }

    return pieces
  })
}

export function coordinateToEngineSquare(coordinate: Coordinate) {
  return `${files[coordinate.file] ?? '?'}${9 - coordinate.rank}`
}

export function engineSquareToCoordinate(square: string): Coordinate | null {
  const file = files.indexOf(square[0] ?? '')
  const rank = Number(square[1])

  if (file < 0 || Number.isNaN(rank) || rank < 0 || rank > 9) return null
  return { file, rank: 9 - rank }
}

export function lessonMoveToEngineNotation(move: { from: Coordinate; to: Coordinate }) {
  return `${coordinateToEngineSquare(move.from)}${coordinateToEngineSquare(move.to)}`
}

export function engineNotationToMove(notation: string) {
  const from = engineSquareToCoordinate(notation.slice(0, 2))
  const to = engineSquareToCoordinate(notation.slice(2, 4))

  if (!from || !to) return null
  return { from, to, notation }
}

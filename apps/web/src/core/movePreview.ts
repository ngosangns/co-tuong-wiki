import type { LessonLine } from '../api/types'
import type { LessonMove } from './xiangqi'

export interface NextMoveTarget {
  lineId: string
  move: LessonMove
}

export function moveSignature(move: LessonMove) {
  return `${move.side}:${move.from.file},${move.from.rank}:${move.to.file},${move.to.rank}`
}

export function lineStartKey(line: LessonLine, lessonInitialFen?: string) {
  return (line.initialFen ?? lessonInitialFen ?? 'standard').trim()
}

function lineMatchesPrefix(line: LessonLine, prefixMoves: LessonMove[]) {
  const moves = line.moves ?? []
  return prefixMoves.every((move, index) => {
    const candidate = moves[index]
    return candidate ? moveSignature(candidate) === moveSignature(move) : false
  })
}

export function nextMoveTargetsForActiveNode(input: {
  lines: LessonLine[]
  activeLine?: LessonLine
  activeMoveIndex: number
  lessonInitialFen?: string
}) {
  const activeLine = input.activeLine
  if (!activeLine) return []

  const activeStartKey = lineStartKey(activeLine, input.lessonInitialFen)
  const activePrefix = (activeLine.moves ?? []).slice(0, input.activeMoveIndex)
  const targetsBySignature = new Map<string, NextMoveTarget>()

  input.lines.forEach((line) => {
    if (lineStartKey(line, input.lessonInitialFen) !== activeStartKey || !lineMatchesPrefix(line, activePrefix)) return

    const nextMove = line.moves?.[input.activeMoveIndex]
    if (!nextMove) return

    const signature = moveSignature(nextMove)
    const existingTarget = targetsBySignature.get(signature)
    if (existingTarget && line.id !== activeLine.id) return

    targetsBySignature.set(signature, { lineId: line.id, move: nextMove })
  })

  return Array.from(targetsBySignature.values())
}

export function nextMovesForActiveNode(input: Parameters<typeof nextMoveTargetsForActiveNode>[0]) {
  return nextMoveTargetsForActiveNode(input).map((target) => target.move)
}

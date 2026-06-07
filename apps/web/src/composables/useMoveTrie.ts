import { computed, type ComputedRef } from 'vue'
import type { LessonLine } from '../api/types'
import type { BoardState, LessonMove, PieceKind, Side } from '../core/xiangqi'
import { applyMove, initialBoard, pieceAt } from '../core/xiangqi'
import { boardFromXiangqiFen } from '../engine/fen'

export interface MoveGraphPathNode {
  line: LessonLine
  move: LessonMove
  moveIndex: number
  boardBefore: BoardState | null
  signature: string
}

export interface MoveTrieNode {
  id: string
  lineId: string
  lineIds: Set<string>
  children: Map<string, MoveTrieNode>
  move?: LessonMove
  moveIndex?: number
  boardBefore?: BoardState | null
}

export type MoveNodeKind = 'start' | 'move'
export type MoveNodeState = 'start' | 'active' | 'past' | 'future'

export interface SerializedTrieNode {
  id: string
  lineId: string
  lineIds: string[]
  move?: LessonMove
  moveIndex?: number
  boardBefore?: BoardState | null
  nodeKind: MoveNodeKind
  state: MoveNodeState
  shortLabel: string
  longLabel: string
  children: SerializedTrieNode[]
}

const DEFAULT_XIANGQI_FEN = 'rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1'
const FILES = 'abcdefghi'

const pieceNames: Record<PieceKind, string> = {
  general: 'Tướng',
  advisor: 'Sĩ',
  elephant: 'Tượng',
  horse: 'Mã',
  chariot: 'Xe',
  cannon: 'Pháo',
  soldier: 'Tốt',
}

export function moveSignature(move: LessonMove): string {
  return `${move.side}:${move.from.file},${move.from.rank}:${move.to.file},${move.to.rank}`
}

export function lineInitialFen(line: LessonLine, lessonInitialFen?: string): string {
  return (line.initialFen ?? lessonInitialFen ?? '').trim()
}

export function startKeyForFen(fen: string): string {
  return !fen || fen === DEFAULT_XIANGQI_FEN ? 'standard' : fen
}

export function startNodeId(startKey: string): string {
  return `start:${encodeURIComponent(startKey)}`
}

export function startNodeIdForLine(line: LessonLine, lessonInitialFen?: string): string {
  return startNodeId(startKeyForFen(lineInitialFen(line, lessonInitialFen)))
}

export function moveNodeId(parentId: string, signature: string): string {
  return `${parentId}/move:${encodeURIComponent(signature)}`
}

function sideFileNumber(side: Side, file: number): number {
  return side === 'red' ? file + 1 : 9 - file
}

function directionLabel(side: Side, fromRank: number, toRank: number): 'tiến' | 'thoái' | 'bình' {
  if (fromRank === toRank) return 'bình'
  const isForward = side === 'red' ? toRank < fromRank : toRank > fromRank
  return isForward ? 'tiến' : 'thoái'
}

export function formatParsedMoveNotation(move: LessonMove, boardBefore: BoardState | null): string {
  const piece = boardBefore ? pieceAt(boardBefore, move.from) : null
  if (!piece) return `(${move.from.file},${move.from.rank})→(${move.to.file},${move.to.rank})`

  const verb = directionLabel(move.side, move.from.rank, move.to.rank)
  const fromFile = sideFileNumber(move.side, move.from.file)
  const toFile = sideFileNumber(move.side, move.to.file)
  const distance = Math.abs(move.to.rank - move.from.rank)
  const usesStepCount =
    move.from.file === move.to.file &&
    verb !== 'bình' &&
    ['chariot', 'cannon', 'general', 'soldier'].includes(piece.kind)
  const target = usesStepCount ? distance : toFile

  return `${pieceNames[piece.kind]} ${fromFile} ${verb} ${target}`
}

export function formatMoveNotation(
  move: LessonMove,
  moveIndex: number,
  boardBefore: BoardState | null,
): string {
  const moveNumber = Math.floor(moveIndex / 2) + 1
  const notation = formatParsedMoveNotation(move, boardBefore)
  return moveIndex % 2 === 0 ? `${moveNumber}. ${notation}` : `${moveNumber}... ${notation}`
}

export function lineMoves(line: LessonLine): LessonMove[] {
  return line.moves ?? []
}

export function buildGraphPaths(lines: LessonLine[], lessonInitialFen?: string): MoveGraphPathNode[][] {
  return lines.map((line) => {
    const initial = lineInitialFen(line, lessonInitialFen)
      ? boardFromXiangqiFen(lineInitialFen(line, lessonInitialFen))
      : initialBoard
    let board: BoardState = initial

    return lineMoves(line).map<MoveGraphPathNode>((move, index) => {
      const boardBefore = board
      try {
        board = applyMove(board, move)
      } catch {
        board = boardBefore
      }

      return {
        line,
        move,
        moveIndex: index,
        boardBefore,
        signature: moveSignature(move),
      }
    })
  })
}

export function buildMoveTrie(
  lines: LessonLine[],
  lessonInitialFen?: string,
): {
  startGroups: Map<string, { root: MoveTrieNode }>
  totalMoves: number
} {
  const startGroups = new Map<string, { root: MoveTrieNode }>()
  let totalMoves = 0

  for (const line of lines) {
    const startKey = startKeyForFen(lineInitialFen(line, lessonInitialFen))
    let group = startGroups.get(startKey)
    if (!group) {
      group = {
        root: {
          id: startNodeId(startKey),
          lineId: '',
          lineIds: new Set<string>(),
          children: new Map<string, MoveTrieNode>(),
        },
      }
      startGroups.set(startKey, group)
    }
    group.root.lineIds.add(line.id)
    if (!group.root.lineId) group.root.lineId = line.id
  }

  for (const path of buildGraphPaths(lines, lessonInitialFen)) {
    const line = path[0]?.line
    if (!line) continue
    const group = startGroups.get(startKeyForFen(lineInitialFen(line, lessonInitialFen)))
    if (!group) continue

    let parent = group.root
    for (const pathNode of path) {
      let child = parent.children.get(pathNode.signature)
      if (!child) {
        child = {
          id: moveNodeId(parent.id, pathNode.signature),
          lineId: pathNode.line.id,
          lineIds: new Set<string>(),
          children: new Map<string, MoveTrieNode>(),
          move: pathNode.move,
          moveIndex: pathNode.moveIndex,
          boardBefore: pathNode.boardBefore,
        }
        parent.children.set(pathNode.signature, child)
      }
      child.lineIds.add(pathNode.line.id)
      if (!child.lineId) child.lineId = pathNode.line.id
      parent = child
      totalMoves++
    }
  }

  return { startGroups, totalMoves }
}

export function serializeTrie(startGroups: Map<string, { root: MoveTrieNode }>): SerializedTrieNode[] {
  const out: SerializedTrieNode[] = []

  function visit(node: MoveTrieNode, startKey: string): SerializedTrieNode {
    const lineIds = Array.from(node.lineIds)
    const longLabel =
      node.move && node.moveIndex !== undefined
        ? `${formatMoveNotation(node.move, node.moveIndex, node.boardBefore ?? null)}${lineIds.length > 1 ? ` · ${lineIds.length} biến` : ''}`
        : startKey === 'standard'
          ? `Vị trí chuẩn · ${lineIds.length} biến`
          : `Vị trí FEN · ${lineIds.length} biến`

    const shortLabel =
      node.move && node.moveIndex !== undefined
        ? formatParsedMoveNotation(node.move, node.boardBefore ?? null)
        : startKey === 'standard'
          ? 'Bắt đầu'
          : 'FEN'

    return {
      id: node.id,
      lineId: node.lineId,
      lineIds,
      move: node.move,
      moveIndex: node.moveIndex,
      boardBefore: node.boardBefore ?? null,
      nodeKind: node.move ? 'move' : 'start',
      state: node.move ? 'future' : 'start',
      shortLabel,
      longLabel,
      children: Array.from(node.children.values()).map((child) => visit(child, startKey)),
    }
  }

  startGroups.forEach((group, key) => {
    out.push(visit(group.root, key))
  })

  return out
}

export function useMoveTrie(
  lines: ComputedRef<LessonLine[]> | (() => LessonLine[]),
  initialFen: ComputedRef<string | undefined> | (() => string | undefined),
) {
  const source = typeof lines === 'function' ? lines : () => lines.value
  const initialSource = typeof initialFen === 'function' ? initialFen : () => initialFen.value

  const graphPaths = computed(() => buildGraphPaths(source(), initialSource()))
  const trie = computed(() => buildMoveTrie(source(), initialSource()))
  const serialized = computed(() => serializeTrie(trie.value.startGroups))
  const totalMoves = computed(() => trie.value.totalMoves)
  const lineCount = computed(() => source().length)

  return {
    graphPaths,
    trie,
    serialized,
    totalMoves,
    lineCount,
  }
}

export function buildActiveNodeId(
  lines: LessonLine[],
  activeLineId: string,
  activeMoveIndex: number,
  initialFen?: string,
): string {
  const line = lines.find((item) => item.id === activeLineId)
  if (!line) return ''
  if (activeMoveIndex <= 0) return startNodeIdForLine(line, initialFen)

  return lineMoves(line)
    .slice(0, activeMoveIndex)
    .reduce((nodeId, move) => moveNodeId(nodeId, moveSignature(move)), startNodeIdForLine(line, initialFen))
}

export function findActivePath(
  graphPaths: MoveGraphPathNode[][],
  activeLineId: string,
  activeMoveIndex: number,
): MoveGraphPathNode[] | null {
  const path = graphPaths.find((candidate) => candidate[0]?.line.id === activeLineId)
  if (!path) return null
  return path.slice(0, activeMoveIndex)
}

export function engineSquareLabel(file: number, rank: number): string {
  return `${FILES[file]}${9 - rank}`
}

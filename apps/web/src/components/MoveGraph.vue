<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import type { LessonLine } from '../api/types'
import { applyMove, initialBoard, pieceAt } from '../core/xiangqi'
import type { BoardState, LessonMove, PieceKind, Side } from '../core/xiangqi'
import { boardFromXiangqiFen } from '../engine/fen'

const graphLayout = {
  nodeWidth: 132,
  nodeHeight: 66,
  columnGap: 48,
  rowGap: 58,
  padding: 22,
  labelWidth: 112,
}

const props = defineProps<{
  lines: LessonLine[]
  activeLineId: string
  activeMoveIndex: number
  initialFen?: string
}>()

const emit = defineEmits<{
  selectLine: [lineId: string]
  selectMove: [lineId: string, moveIndex: number]
}>()

const graphRoot = ref<HTMLElement | null>(null)

const activeLine = computed(() => props.lines.find((item) => item.id === props.activeLineId))
const totalMoveCount = computed(() => props.lines.reduce((count, line) => count + line.moves.length, 0))
const activeMove = computed(() => activeLine.value?.moves[props.activeMoveIndex - 1])

const sharedPrefixLength = computed(() => {
  const [firstLine, ...remainingLines] = props.lines
  if (!firstLine || remainingLines.length === 0) return 0

  let prefixLength = firstLine.moves.length
  for (const line of remainingLines) {
    let index = 0
    while (index < prefixLength && firstLine.moves[index]?.id === line.moves[index]?.id) {
      index += 1
    }
    prefixLength = index
  }

  return prefixLength
})

const trunkLine = computed(() => props.lines[0])
const trunkMoves = computed(() => trunkLine.value?.moves.slice(0, sharedPrefixLength.value) ?? [])
const branchLines = computed(() => props.lines)
const graphStats = computed(() => `${props.lines.length} biến, ${totalMoveCount.value} nước`)
const graphRowOffset = computed(() => (sharedPrefixLength.value > 0 ? 1 : 0))
const maxMoveCount = computed(() => Math.max(1, ...props.lines.map((line) => line.moves.length)))
const graphMapWidth = computed(
  () =>
    graphLayout.labelWidth +
    graphLayout.padding * 2 +
    maxMoveCount.value * graphLayout.nodeWidth +
    Math.max(0, maxMoveCount.value - 1) * graphLayout.columnGap,
)
const graphMapHeight = computed(() => {
  const rows = Math.max(1, props.lines.length + graphRowOffset.value)
  return graphLayout.padding * 2 + rows * graphLayout.nodeHeight + Math.max(0, rows - 1) * graphLayout.rowGap
})
const graphMapStyle = computed(() => ({
  width: `${graphMapWidth.value}px`,
  height: `${graphMapHeight.value}px`,
}))

interface GraphNode {
  key: string
  line: LessonLine
  move: LessonMove
  moveIndex: number
  boardBefore: BoardState | null
  row: number
  x: number
  y: number
  state: string
  isShared: boolean
  lineIds: string[]
}

interface GraphEdge {
  key: string
  x1: number
  y1: number
  x2: number
  y2: number
  state: string
  lineIds: string[]
}

interface GraphPathNode {
  key: string
  line: LessonLine
  move: LessonMove
  moveIndex: number
  boardBefore: BoardState | null
  lineIds: string[]
}

function nodeX(moveIndex: number) {
  return graphLayout.labelWidth + graphLayout.padding + moveIndex * (graphLayout.nodeWidth + graphLayout.columnGap)
}

function nodeY(row: number) {
  return graphLayout.padding + row * (graphLayout.nodeHeight + graphLayout.rowGap)
}

function nodeCenterX(moveIndex: number) {
  return nodeX(moveIndex) + graphLayout.nodeWidth / 2
}

function nodeCenterY(row: number) {
  return nodeY(row) + graphLayout.nodeHeight / 2
}

function stepState(line: LessonLine, index: number) {
  const step = index + 1
  const activeMove = activeLine.value?.moves[index]
  const isSameMove = activeMove?.id === line.moves[index]?.id

  if (props.activeLineId === line.id && props.activeMoveIndex === step) return 'active'
  if (isSameMove && props.activeMoveIndex === step) return 'active'
  if (props.activeLineId === line.id && props.activeMoveIndex > step) return 'past'
  if (isSameMove && props.activeMoveIndex > step) return 'past'
  return 'future'
}

function moveGraphKey(move: LessonMove, moveIndex: number) {
  return [moveIndex, move.side, move.from.file, move.from.rank, move.to.file, move.to.rank, move.id].join(':')
}

const pieceNames: Record<PieceKind, string> = {
  general: 'Tướng',
  advisor: 'Sĩ',
  elephant: 'Tượng',
  horse: 'Mã',
  chariot: 'Xe',
  cannon: 'Pháo',
  soldier: 'Tốt',
}

function sideFileNumber(side: Side, file: number) {
  return side === 'red' ? file + 1 : 9 - file
}

function directionLabel(side: Side, fromRank: number, toRank: number) {
  if (fromRank === toRank) return 'bình'
  const isForward = side === 'red' ? toRank < fromRank : toRank > fromRank
  return isForward ? 'tiến' : 'thoái'
}

function formatParsedMoveNotation(move: LessonMove, boardBefore: BoardState | null) {
  const piece = boardBefore ? pieceAt(boardBefore, move.from) : null
  if (!piece) return `(${move.from.file},${move.from.rank}) -> (${move.to.file},${move.to.rank})`

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

function formatMoveNotation(move: LessonMove, moveIndex: number, boardBefore: BoardState | null) {
  const moveNumber = Math.floor(moveIndex / 2) + 1
  const notation = formatParsedMoveNotation(move, boardBefore)
  return moveIndex % 2 === 0 ? `${moveNumber}. ${notation}` : `${moveNumber}... ${notation}`
}

const graphPaths = computed(() => {
  const membership = new Map<string, Set<string>>()

  props.lines.forEach((line) => {
    line.moves.forEach((move, index) => {
      const key = moveGraphKey(move, index)
      const lineIds = membership.get(key) ?? new Set<string>()
      lineIds.add(line.id)
      membership.set(key, lineIds)
    })
  })

  const firstBoard = props.initialFen ? boardFromXiangqiFen(props.initialFen) : initialBoard

  return props.lines.map((line) => {
    let board = firstBoard

    return line.moves.map<GraphPathNode>((move, index) => {
      const key = moveGraphKey(move, index)
      const boardBefore = board
      try {
        board = applyMove(board, move)
      } catch {
        board = boardBefore
      }

      return {
        key,
        line,
        move,
        moveIndex: index,
        boardBefore,
        lineIds: Array.from(membership.get(key) ?? []),
      }
    })
  })
})

const graphNodes = computed<GraphNode[]>(() => {
  const nodes: GraphNode[] = []
  const [sharedPath] = graphPaths.value

  if (sharedPath) {
    sharedPath.slice(0, sharedPrefixLength.value).forEach((pathNode, index) => {
      nodes.push({
        key: `shared-${pathNode.key}`,
        line: pathNode.line,
        move: pathNode.move,
        moveIndex: index,
        boardBefore: pathNode.boardBefore,
        row: 0,
        x: nodeX(index),
        y: nodeY(0),
        state: stepState(pathNode.line, index),
        isShared: true,
        lineIds: pathNode.lineIds,
      })
    })
  }

  graphPaths.value.forEach((path, lineIndex) => {
    const row = graphRowOffset.value + lineIndex
    path.slice(sharedPrefixLength.value).forEach((pathNode, branchIndex) => {
      const moveIndex = sharedPrefixLength.value + branchIndex
      nodes.push({
        key: `${pathNode.line.id}-${pathNode.key}`,
        line: pathNode.line,
        move: pathNode.move,
        moveIndex,
        boardBefore: pathNode.boardBefore,
        row,
        x: nodeX(moveIndex),
        y: nodeY(row),
        state: stepState(pathNode.line, moveIndex),
        isShared: false,
        lineIds: pathNode.lineIds,
      })
    })
  })

  return nodes
})

const activeBoardBeforeMove = computed(() => {
  if (!activeLine.value || props.activeMoveIndex <= 0) return null
  const activePath = graphPaths.value.find((path) => path[0]?.line.id === activeLine.value?.id)
  return activePath?.[props.activeMoveIndex - 1]?.boardBefore ?? null
})

const graphEdges = computed<GraphEdge[]>(() => {
  const edges: GraphEdge[] = []
  const hasSharedTrunk = sharedPrefixLength.value > 0
  const [sharedPath] = graphPaths.value

  if (sharedPath) {
    sharedPath.slice(1, sharedPrefixLength.value).forEach((pathNode, index) => {
      const moveIndex = index + 1
      edges.push({
        key: `shared-edge-${pathNode.key}`,
        x1: nodeCenterX(moveIndex - 1),
        y1: nodeCenterY(0),
        x2: nodeCenterX(moveIndex),
        y2: nodeCenterY(0),
        state: stepState(pathNode.line, moveIndex),
        lineIds: pathNode.lineIds,
      })
    })
  }

  graphPaths.value.forEach((path, lineIndex) => {
    const row = graphRowOffset.value + lineIndex
    path.slice(sharedPrefixLength.value).forEach((pathNode, branchIndex) => {
      const moveIndex = sharedPrefixLength.value + branchIndex
      const currentState = stepState(pathNode.line, moveIndex)
      const fromShared = branchIndex === 0 && hasSharedTrunk
      if (fromShared) {
        edges.push({
          key: `${pathNode.line.id}-fork-${pathNode.key}`,
          x1: nodeCenterX(sharedPrefixLength.value - 1),
          y1: nodeCenterY(0),
          x2: nodeCenterX(moveIndex),
          y2: nodeCenterY(row),
          state: currentState,
          lineIds: pathNode.lineIds,
        })
        return
      }

      if (branchIndex > 0) {
        edges.push({
          key: `${pathNode.line.id}-edge-${pathNode.key}`,
          x1: nodeCenterX(moveIndex - 1),
          y1: nodeCenterY(row),
          x2: nodeCenterX(moveIndex),
          y2: nodeCenterY(row),
          state: currentState,
          lineIds: pathNode.lineIds,
        })
      }
    })
  })

  return edges
})

function selectGraphNode(node: GraphNode) {
  // Shared graph nodes keep the current branch whenever possible; branch nodes own their line.
  const lineId = node.isShared && node.lineIds.includes(props.activeLineId) ? props.activeLineId : node.line.id
  emit('selectMove', lineId, node.moveIndex + 1)
}

function lineLabelStyle(lineIndex: number) {
  return {
    top: `${nodeY(graphRowOffset.value + lineIndex)}px`,
  }
}

function moveSideLabel(move?: LessonMove) {
  if (!move) return 'Chưa chọn nước'
  return move.side === 'red' ? 'Đỏ' : 'Đen'
}

async function keepActiveNodeVisible() {
  // The graph updates classes after the playhead changes, so scroll after Vue has flushed.
  await nextTick()
  graphRoot.value?.querySelector('.graph-step.active')?.scrollIntoView({
    behavior: 'smooth',
    block: 'nearest',
    inline: 'center',
  })
}

watch(() => [props.activeLineId, props.activeMoveIndex], keepActiveNodeVisible)
</script>

<template>
  <section ref="graphRoot" class="move-graph graph-shell" aria-label="Cây nước đi">
    <header class="graph-toolbar">
      <div>
        <h3>Cây nước đi</h3>
        <p>{{ graphStats }}</p>
      </div>
    </header>

    <div class="graph-layout">
      <div class="graph-canvas">
        <div class="graph-map" :style="graphMapStyle">
          <div v-if="trunkLine && trunkMoves.length" class="graph-branch-label graph-shared-label">Chung</div>

          <button
            v-for="line in branchLines"
            :key="line.id"
            type="button"
            class="graph-branch-label graph-line-label"
            :class="{ active: activeLineId === line.id }"
            :style="lineLabelStyle(props.lines.findIndex((item) => item.id === line.id))"
            @click="emit('selectLine', line.id)"
          >
            <span>{{ line.title }}</span>
          </button>

          <svg class="graph-edges" :viewBox="`0 0 ${graphMapWidth} ${graphMapHeight}`" aria-hidden="true">
            <line
              v-for="edge in graphEdges"
              :key="edge.key"
              class="graph-edge"
              :class="edge.state"
              :x1="edge.x1"
              :y1="edge.y1"
              :x2="edge.x2"
              :y2="edge.y2"
            />
          </svg>

          <button
            v-for="node in graphNodes"
            :key="node.key"
            type="button"
            class="graph-step"
            :class="[node.state, { shared: node.isShared }]"
            :style="{ left: `${node.x}px`, top: `${node.y}px` }"
            @click="selectGraphNode(node)"
          >
            <span class="graph-step-index">{{ node.moveIndex + 1 }}</span>
            <span class="graph-step-copy">
              <strong>{{ formatMoveNotation(node.move, node.moveIndex, node.boardBefore) }}</strong>
              <small>{{ node.move.comment }}</small>
            </span>
          </button>
        </div>
      </div>

      <aside class="graph-details">
        <span>{{ moveSideLabel(activeMove) }}</span>
        <strong>{{ activeMove ? formatMoveNotation(activeMove, activeMoveIndex - 1, activeBoardBeforeMove) : 'Bắt đầu' }}</strong>
        <p>{{ activeMove?.comment ?? 'Chọn một node hoặc bấm Nước kế để theo dõi biến hóa.' }}</p>
        <button v-if="activeLine" type="button" @click="emit('selectLine', activeLine.id)">
          {{ activeLine.title }}
        </button>
      </aside>
    </div>
  </section>
</template>

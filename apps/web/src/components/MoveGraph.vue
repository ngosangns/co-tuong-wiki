<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import type { LessonLine } from '../api/types'
import { applyMove, initialBoard, pieceAt } from '../core/xiangqi'
import type { BoardState, LessonMove, PieceKind, Side } from '../core/xiangqi'
import { boardFromXiangqiFen } from '../engine/fen'
import {
  renderTreeGraph,
  type TreeGraphData,
  type TreeGraphNode,
  type TreeGraphRenderer,
} from '../graph/treeGraph'

const props = defineProps<{
  lines: LessonLine[]
  activeLineId: string
  activeMoveIndex: number
  initialFen?: string
  title?: string
  isCollapsible?: boolean
  isCollapsed?: boolean
}>()

const emit = defineEmits<{
  selectLine: [lineId: string]
  selectMove: [lineId: string, moveIndex: number]
  toggleCollapse: []
}>()

interface MoveGraphNode extends TreeGraphNode {
  nodeKind: 'start' | 'move'
  lineId: string
  lineIds?: string[]
  moveIndex?: number
  state: 'start' | 'line' | 'active' | 'past' | 'future'
}

interface GraphPathNode {
  line: LessonLine
  move: LessonMove
  moveIndex: number
  boardBefore: BoardState | null
  signature: string
}

interface TrieNode {
  id: string
  lineId: string
  lineIds: Set<string>
  children: Map<string, TrieNode>
  move?: LessonMove
  moveIndex?: number
  boardBefore?: BoardState | null
}

const defaultXiangqiFEN = 'rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1'
const graphCanvas = ref<HTMLElement | null>(null)
const renderer = ref<TreeGraphRenderer | null>(null)
const selectedNodeId = ref('')
const isZoomed = ref(false)

const activeLine = computed(() => props.lines.find((item) => item.id === props.activeLineId))
const activeMove = computed(() =>
  activeLine.value ? lineMoves(activeLine.value)[props.activeMoveIndex - 1] : undefined,
)
const totalMoveCount = computed(() => props.lines.reduce((count, line) => count + lineMoveCount(line), 0))
const graphStats = computed(() => `${props.lines.length} biến, ${totalMoveCount.value} nước`)
function lineMoves(line: LessonLine) {
  return line.moves ?? []
}

function lineMoveCount(line: LessonLine) {
  return line.moveCount ?? lineMoves(line).length
}

function lineInitialFen(line: LessonLine) {
  return (line.initialFen ?? props.initialFen ?? '').trim()
}

function startKeyForFen(fen: string) {
  return !fen || fen === defaultXiangqiFEN ? 'standard' : fen
}

function startNodeId(startKey: string) {
  return `start:${encodeURIComponent(startKey)}`
}

function startNodeIdForLine(line: LessonLine) {
  return startNodeId(startKeyForFen(lineInitialFen(line)))
}

function startNodeIdForLineId(lineId: string) {
  const line = props.lines.find((item) => item.id === lineId)
  return line ? startNodeIdForLine(line) : ''
}

function moveSignature(move: LessonMove) {
  return `${move.side}:${move.from.file},${move.from.rank}:${move.to.file},${move.to.rank}`
}

function moveNodeId(parentId: string, signature: string) {
  return `${parentId}/move:${encodeURIComponent(signature)}`
}

function activeGraphNodeId() {
  if (!props.activeLineId) return ''
  const line = props.lines.find((item) => item.id === props.activeLineId)
  if (!line || props.activeMoveIndex <= 0) return startNodeIdForLineId(props.activeLineId)

  return lineMoves(line)
    .slice(0, props.activeMoveIndex)
    .reduce((nodeId, move) => moveNodeId(nodeId, moveSignature(move)), startNodeIdForLine(line))
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

const graphPaths = computed(() =>
  props.lines.map((line) => {
    let board = lineInitialFen(line) ? boardFromXiangqiFen(lineInitialFen(line)) : initialBoard

    return lineMoves(line).map<GraphPathNode>((move, index) => {
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
  }),
)

const graphData = computed<TreeGraphData>(() => {
  const nodes: MoveGraphNode[] = []
  const links: TreeGraphData['links'] = []
  const startGroups = new Map<string, { lines: LessonLine[]; root: TrieNode }>()

  props.lines.forEach((line) => {
    const key = startKeyForFen(lineInitialFen(line))
    const group = startGroups.get(key) ?? {
      lines: [],
      root: {
        id: startNodeId(key),
        lineId: '',
        lineIds: new Set<string>(),
        children: new Map<string, TrieNode>(),
      },
    }
    group.lines.push(line)
    group.root.lineIds.add(line.id)
    group.root.lineId = group.root.lineId || line.id
    startGroups.set(key, group)
  })

  graphPaths.value.forEach((path) => {
    const line = path[0]?.line
    if (!line) return
    const group = startGroups.get(startKeyForFen(lineInitialFen(line)))
    if (!group) return

    let parent = group.root
    path.forEach((pathNode) => {
      let child = parent.children.get(pathNode.signature)
      if (!child) {
        child = {
          id: moveNodeId(parent.id, pathNode.signature),
          lineId: pathNode.line.id,
          lineIds: new Set<string>(),
          children: new Map<string, TrieNode>(),
          move: pathNode.move,
          moveIndex: pathNode.moveIndex,
          boardBefore: pathNode.boardBefore,
        }
        parent.children.set(pathNode.signature, child)
      }
      child.lineIds.add(pathNode.line.id)
      if (!child.lineId) child.lineId = pathNode.line.id
      parent = child
    })
  })

  function addTrieNode(node: TrieNode, startKey: string, parent?: TrieNode) {
    const lineIds = Array.from(node.lineIds)
    const label =
      node.move && node.moveIndex !== undefined
        ? `${formatMoveNotation(node.move, node.moveIndex, node.boardBefore ?? null)}${lineIds.length > 1 ? ` · ${lineIds.length} biến` : ''}`
        : startKey === 'standard'
          ? `Vị trí chuẩn · ${lineIds.length} biến`
          : `Vị trí FEN · ${lineIds.length} biến`

    nodes.push({
      id: node.id,
      label,
      nodeKind: node.move ? 'move' : 'start',
      lineId: node.lineId,
      lineIds,
      moveIndex: node.moveIndex,
      state: node.move ? 'future' : 'start',
    })

    if (parent) {
      links.push({
        source: parent.id,
        target: node.id,
        type: 'next',
      })
    }

    node.children.forEach((child) => addTrieNode(child, startKey, node))
  }

  startGroups.forEach((group, key) => {
    addTrieNode(group.root, key)
  })

  return { nodes, links }
})

const activeNodeId = computed(activeGraphNodeId)
const graphNodeStates = computed(() =>
  Object.fromEntries(
    graphData.value.nodes.map((node) => {
      const graphNode = node as MoveGraphNode
      const containsActiveLine = Boolean(graphNode.lineIds?.includes(props.activeLineId))
      const state =
        graphNode.moveIndex === undefined
          ? containsActiveLine && props.activeMoveIndex <= 0
            ? 'active'
            : 'start'
          : containsActiveLine && props.activeMoveIndex === graphNode.moveIndex + 1
            ? 'active'
            : containsActiveLine && props.activeMoveIndex > graphNode.moveIndex + 1
              ? 'past'
              : 'future'

      return [graphNode.id, state]
    }),
  ),
)
const activeBoardBeforeMove = computed(() => {
  if (!activeLine.value || props.activeMoveIndex <= 0) return null
  const activePath = graphPaths.value.find((path) => path[0]?.line.id === activeLine.value?.id)
  return activePath?.[props.activeMoveIndex - 1]?.boardBefore ?? null
})

const selectedNode = computed(
  () =>
    graphData.value.nodes.find((node) => node.id === (selectedNodeId.value || activeNodeId.value)) as
      | MoveGraphNode
      | undefined,
)
const selectedLine = computed(
  () => props.lines.find((line) => line.id === selectedNode.value?.lineId) ?? activeLine.value,
)
const selectedMove = computed(() => {
  const node = selectedNode.value
  if (!node || node.nodeKind !== 'move') return activeMove.value
  const line = props.lines.find((item) => item.id === node.lineId)
  return line ? lineMoves(line)[node.moveIndex ?? -1] : undefined
})
const selectedMoveIndex = computed(() =>
  selectedNode.value?.nodeKind === 'move' ? (selectedNode.value.moveIndex ?? -1) : props.activeMoveIndex - 1,
)
const selectedBoardBeforeMove = computed(() => {
  const node = selectedNode.value
  if (!node || node.nodeKind !== 'move') return activeBoardBeforeMove.value
  const path = graphPaths.value.find((item) => item[0]?.line.id === node.lineId)
  return path?.[node.moveIndex ?? -1]?.boardBefore ?? null
})

function moveSideLabel(move?: LessonMove) {
  if (!move) return 'Chưa chọn nước'
  return move.side === 'red' ? 'Đỏ' : 'Đen'
}

function nodeColor(node: TreeGraphNode) {
  if (node.state === 'active') return '#f06c56'
  if (node.state === 'past') return '#70b8a7'
  if (node.state === 'start') return '#d9a441'
  if (node.state === 'line') return '#8d7359'
  return '#c7bda9'
}

function edgeColor(type: string) {
  return type === 'active' ? '#70b8a7' : '#8d7359'
}

function selectGraphNode(node: TreeGraphNode) {
  const graphNode = node as MoveGraphNode
  selectedNodeId.value = graphNode.id
  if (graphNode.nodeKind === 'start') {
    emit(
      'selectMove',
      graphNode.lineIds?.includes(props.activeLineId) ? props.activeLineId : graphNode.lineId,
      0,
    )
    return
  }
  emit(
    'selectMove',
    graphNode.lineIds?.includes(props.activeLineId) ? props.activeLineId : graphNode.lineId,
    (graphNode.moveIndex ?? 0) + 1,
  )
}

function clearSelection() {
  selectedNodeId.value = ''
}

function destroyGraph() {
  renderer.value?.kill()
  renderer.value = null
}

function fitGraph() {
  renderer.value?.fit()
}

function resetGraph() {
  renderer.value?.reset()
}

async function renderGraph() {
  await nextTick()
  if (!graphCanvas.value) return
  destroyGraph()
  graphCanvas.value.innerHTML = ''
  isZoomed.value = false
  renderer.value = renderTreeGraph({
    container: graphCanvas.value,
    graph: graphData.value,
    selectedId: activeNodeId.value,
    nodeColor,
    edgeColor,
    labelColor: '#f5efe2',
    onSelectNode: selectGraphNode,
    onClearSelection: clearSelection,
    onZoomChange: (zoomed) => {
      isZoomed.value = zoomed
    },
  })
  renderer.value.setStates(graphNodeStates.value)
}

watch(graphData, renderGraph, { immediate: true })
watch(activeNodeId, (nodeId) => renderer.value?.setSelected(nodeId))
watch(graphNodeStates, (states) => renderer.value?.setStates(states))
watch(
  () => props.isCollapsed,
  (isCollapsed) => {
    if (isCollapsed) {
      destroyGraph()
      return
    }

    void renderGraph()
  },
)

onMounted(renderGraph)
onUnmounted(destroyGraph)
</script>

<template>
  <section class="move-graph graph-shell" aria-label="Cây nước đi">
    <header class="graph-toolbar">
      <slot name="toolbar" :title="title ?? 'Cây nước đi'" :stats="graphStats" :fit="fitGraph">
        <button
          v-if="isCollapsible"
          type="button"
          class="graph-title-button"
          :aria-expanded="!isCollapsed"
          @click="emit('toggleCollapse')"
        >
          <span class="graph-title-copy">
            <h3>{{ title ?? 'Cây nước đi' }}</h3>
            <p>{{ graphStats }}</p>
          </span>
          <span class="graph-collapse-label">{{ isCollapsed ? 'Mở' : 'Đóng' }}</span>
        </button>

        <div v-else>
          <h3>{{ title ?? 'Cây nước đi' }}</h3>
          <p>{{ graphStats }}</p>
        </div>

        <button
          v-if="!isCollapsed"
          type="button"
          class="graph-fit-button"
          title="Canh giữa graph"
          aria-label="Canh giữa graph"
          @click="fitGraph"
        >
          Fit
        </button>
        <button
          v-if="!isCollapsed && isZoomed"
          type="button"
          class="graph-reset-button"
          title="Đặt lại zoom"
          aria-label="Đặt lại zoom"
          @click="resetGraph"
        >
          Reset
        </button>
      </slot>
    </header>

    <div v-if="!isCollapsed" class="graph-layout">
      <div ref="graphCanvas" class="graph-canvas graph-tree-canvas" role="img" aria-label="Move tree"></div>

      <aside class="graph-details">
        <span>{{ moveSideLabel(selectedMove) }}</span>
        <strong>
          {{
            selectedMove
              ? formatMoveNotation(selectedMove, selectedMoveIndex, selectedBoardBeforeMove)
              : 'Bắt đầu'
          }}
        </strong>
        <button v-if="selectedLine" type="button" @click="emit('selectLine', selectedLine.id)">
          {{ selectedLine.title }}
        </button>
      </aside>
    </div>
  </section>
</template>

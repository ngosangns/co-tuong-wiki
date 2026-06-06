<script setup lang="ts">
import { Maximize2 } from '@lucide/vue'
import { computed, nextTick, onMounted, watch } from 'vue'
import { hierarchy, tree } from 'd3-hierarchy'
import type { LessonLine } from '../api/types'
import {
  buildActiveNodeId,
  useMoveTrie,
  type SerializedTrieNode,
} from '../composables/useMoveTrie'

const props = defineProps<{
  lines: LessonLine[]
  activeLineId: string
  activeMoveIndex: number
  initialFen?: string
}>()

const emit = defineEmits<{
  selectMove: [lineId: string, index: number]
  openGraph: []
}>()

const linesSource = computed(() => props.lines)
const initialSource = computed(() => props.initialFen)
const { serialized, totalMoves, lineCount } = useMoveTrie(linesSource, initialSource)

interface PositionedNode {
  id: string
  x: number
  y: number
  label: string
  longLabel: string
  lineId: string
  state: 'active' | 'past' | 'future' | 'start'
}

const MINIMAP_WIDTH = 320
const MINIMAP_HEIGHT = 180
const PADDING = 16

const positions = computed<PositionedNode[]>(() => {
  const roots = serialized.value
  if (!roots.length) return []

  const fakeRoot: SerializedTrieNode = {
    id: '__minimap_root__',
    lineId: '',
    lineIds: [],
    nodeKind: 'start',
    state: 'start',
    shortLabel: '',
    longLabel: '',
    children: roots,
  }

  const root = tree<SerializedTrieNode>().nodeSize([8, 14])(hierarchy(fakeRoot))
  const descendants = root.descendants().filter((node) => node.data.id !== fakeRoot.id)

  return descendants.map((node) => {
    const datum = node.data
    return {
      id: datum.id,
      x: node.y,
      y: node.x,
      label: datum.shortLabel,
      longLabel: datum.longLabel,
      lineId: datum.lineId,
      state: 'future' as const,
    }
  })
})

const positionedLookup = computed(() => {
  const map = new Map<string, PositionedNode>()
  for (const node of positions.value) map.set(node.id, node)
  return map
})

const layoutBounds = computed(() => {
  const nodes = positions.value
  if (!nodes.length) {
    return { minX: 0, minY: 0, width: MINIMAP_WIDTH, height: MINIMAP_HEIGHT }
  }
  const xs = nodes.map((n) => n.x)
  const ys = nodes.map((n) => n.y)
  const minX = Math.min(...xs) - 12
  const maxX = Math.max(...xs) + 12
  const minY = Math.min(...ys) - 12
  const maxY = Math.max(...ys) + 12
  return {
    minX,
    minY,
    width: Math.max(1, maxX - minX),
    height: Math.max(1, maxY - minY),
  }
})

const viewBox = computed(() => {
  const bounds = layoutBounds.value
  const aspect = bounds.width / bounds.height
  const targetAspect = (MINIMAP_WIDTH - PADDING * 2) / (MINIMAP_HEIGHT - PADDING * 2)
  if (aspect > targetAspect) {
    return `${bounds.minX} ${bounds.minY - (bounds.width / targetAspect - bounds.height) / 2} ${bounds.width} ${bounds.width / targetAspect}`
  }
  return `${bounds.minX} ${bounds.minY} ${bounds.width} ${bounds.height}`
})

const activeNodeId = computed(() =>
  buildActiveNodeId(props.lines, props.activeLineId, props.activeMoveIndex, props.initialFen),
)

const activePosition = computed(() => positionedLookup.value.get(activeNodeId.value))

interface NodeStatePosition {
  id: string
  state: 'active' | 'past' | 'future' | 'start'
  x: number
  y: number
  lineId: string
}

const stateDecorations = computed<NodeStatePosition[]>(() =>
  positions.value
    .filter((node) => !node.id.startsWith('__'))
    .map((node) => {
      const containsActiveLine = node.lineId === props.activeLineId
      let state: NodeStatePosition['state'] = 'future'
      if (node.id.startsWith('start:')) {
        state = containsActiveLine && props.activeMoveIndex <= 0 ? 'active' : 'start'
      } else {
        const moveIndex = parseMoveIndexFromId(node.id)
        if (moveIndex === undefined) {
          state = 'future'
        } else if (containsActiveLine && props.activeMoveIndex === moveIndex + 1) {
          state = 'active'
        } else if (containsActiveLine && props.activeMoveIndex > moveIndex + 1) {
          state = 'past'
        } else {
          state = 'future'
        }
      }
      return { id: node.id, state, x: node.x, y: node.y, lineId: node.lineId }
    }),
)

function parseMoveIndexFromId(id: string): number | undefined {
  const matches = id.match(/\/move:/g)
  if (!matches) return undefined
  return matches.length - 1
}

const activePathIds = computed<Set<string>>(() => {
  const set = new Set<string>()
  if (!activeNodeId.value) return set
  set.add(`start:${encodeURIComponent(props.initialFen ?? 'standard')}`)
  let current = `start:${encodeURIComponent(props.initialFen ?? 'standard')}`
  for (let i = 0; i < props.activeMoveIndex; i++) {
    const next = `${current}/move:${i}`
    if (positionedLookup.value.has(next)) {
      set.add(next)
      current = next
    }
  }
  return set
})

const activeEdgeDecorations = computed<Array<{ from: PositionedNode; to: PositionedNode }>>(() => {
  const path = activePathIds.value
  const edges: Array<{ from: PositionedNode; to: PositionedNode }> = []
  for (const id of path) {
    const slashIdx = id.lastIndexOf('/move:')
    if (slashIdx < 0) continue
    const parentId = id.slice(0, slashIdx)
    const from = positionedLookup.value.get(parentId)
    const to = positionedLookup.value.get(id)
    if (from && to) edges.push({ from, to })
  }
  return edges
})

function nodeColor(state: NodeStatePosition['state']): string {
  if (state === 'active') return 'var(--color-accent)'
  if (state === 'past') return 'color-mix(in srgb, var(--color-primary) 64%, transparent)'
  if (state === 'start') return 'var(--color-accent)'
  return 'color-mix(in srgb, var(--color-text) 22%, transparent)'
}

function nodeRadius(state: NodeStatePosition['state']): number {
  if (state === 'active') return 6
  if (state === 'past') return 5
  return 4
}

function onNodeClick(node: NodeStatePosition) {
  if (!props.activeLineId) return
  if (node.state === 'start' || node.state === 'future') return
  const moveIndex = parseMoveIndexFromId(node.id)
  if (moveIndex === undefined) return
  emit('selectMove', props.activeLineId, moveIndex + 1)
}

function onOpenGraph() {
  emit('openGraph')
}

const stats = computed(() => `${lineCount.value} biến · ${totalMoves.value} nước`)

function viewportBox() {
  if (!activePosition.value) return null
  return {
    x: activePosition.value.x - 18,
    y: activePosition.value.y - 18,
    width: 36,
    height: 36,
  }
}

const viewportBoxInViewBox = computed(() => viewportBox())

watch(serialized, () => {
  void nextTick()
})

onMounted(() => {
  void nextTick()
})
</script>

<template>
  <section class="move-minimap" aria-label="Sơ đồ toàn cảnh nước đi">
    <header class="move-minimap-header">
      <div>
        <h4>Sơ đồ toàn cảnh</h4>
        <p>{{ stats }}</p>
      </div>
      <button
        type="button"
        class="move-minimap-open"
        title="Mở graph đầy đủ"
        aria-label="Mở graph đầy đủ"
        @click="onOpenGraph"
      >
        <Maximize2 :size="14" aria-hidden="true" />
      </button>
    </header>
    <div class="move-minimap-canvas">
      <svg
        class="move-minimap-svg"
        :viewBox="viewBox"
        preserveAspectRatio="xMidYMid meet"
        role="img"
        aria-label="Toàn cảnh cây nước đi"
      >
        <g class="move-minimap-edges">
          <line
            v-for="(node, idx) in positions.filter((n) => !n.id.startsWith('__'))"
            :key="`edge-${idx}`"
            :x1="node.x"
            :y1="node.y"
            :x2="node.x + 1"
            :y2="node.y + 1"
            stroke="color-mix(in srgb, var(--color-text) 12%, transparent)"
            stroke-width="0.6"
          />
        </g>
        <g class="move-minimap-active-edges">
          <line
            v-for="(edge, idx) in activeEdgeDecorations"
            :key="`active-edge-${idx}`"
            :x1="edge.from.x"
            :y1="edge.from.y"
            :x2="edge.to.x"
            :y2="edge.to.y"
            stroke="var(--color-accent)"
            stroke-width="1.4"
            stroke-linecap="round"
          />
        </g>
        <g class="move-minimap-nodes">
          <g
            v-for="node in stateDecorations"
            :key="node.id"
            class="move-minimap-node"
            :class="`is-${node.state}`"
            :transform="`translate(${node.x}, ${node.y})`"
          >
            <circle
              :r="nodeRadius(node.state)"
              :fill="nodeColor(node.state)"
              :stroke="node.state === 'active' ? 'var(--color-accent)' : 'transparent'"
              stroke-width="1"
              style="cursor: pointer"
              @click="onNodeClick(node)"
            />
          </g>
        </g>
        <rect
          v-if="viewportBoxInViewBox"
          :x="viewportBoxInViewBox.x"
          :y="viewportBoxInViewBox.y"
          :width="viewportBoxInViewBox.width"
          :height="viewportBoxInViewBox.height"
          fill="none"
          stroke="var(--color-accent)"
          stroke-width="0.8"
          stroke-dasharray="2 2"
          opacity="0.7"
        />
      </svg>
    </div>
  </section>
</template>

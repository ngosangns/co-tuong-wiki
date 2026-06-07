<script setup lang="ts">
import { ChevronRight, Flag } from '@lucide/vue'
import { computed } from 'vue'
import type { LessonLine } from '../api/types'
import {
  buildActiveNodeId,
  findActivePath,
  formatParsedMoveNotation,
  useMoveTrie,
} from '../composables/useMoveTrie'

const props = defineProps<{
  lines: LessonLine[]
  activeLineId: string
  activeMoveIndex: number
  initialFen?: string
}>()

const emit = defineEmits<{
  selectMove: [lineId: string, index: number]
}>()

const linesSource = computed(() => props.lines)
const initialSource = computed(() => props.initialFen)
const { graphPaths } = useMoveTrie(linesSource, initialSource)

const activePath = computed(() => findActivePath(graphPaths.value, props.activeLineId, props.activeMoveIndex))

const activeNodeId = computed(() =>
  buildActiveNodeId(props.lines, props.activeLineId, props.activeMoveIndex, props.initialFen),
)

interface Crumb {
  nodeId: string
  index: number
  label: string
  isStart: boolean
  isActive: boolean
}

const crumbs = computed<Crumb[]>(() => {
  const path = activePath.value ?? []
  const result: Crumb[] = [
    {
      nodeId: `start:${props.initialFen ?? 'standard'}`,
      index: 0,
      label: 'Bắt đầu',
      isStart: true,
      isActive: props.activeMoveIndex === 0,
    },
  ]
  path.forEach((node, idx) => {
    const targetIdx = idx + 1
    result.push({
      nodeId: activeNodeId.value,
      index: targetIdx,
      label: formatParsedMoveNotation(node.move, node.boardBefore),
      isStart: false,
      isActive: targetIdx === props.activeMoveIndex,
    })
  })
  return result
})

function jumpTo(crumb: Crumb) {
  if (crumb.index === 0) {
    emit('selectMove', props.activeLineId, 0)
  } else {
    emit('selectMove', props.activeLineId, crumb.index)
  }
}
</script>

<template>
  <nav class="move-breadcrumb" aria-label="Đường dẫn nước đi">
    <ol class="move-breadcrumb-list">
      <li
        v-for="(crumb, index) in crumbs"
        :key="`${crumb.nodeId}-${crumb.index}-${index}`"
        class="move-breadcrumb-item"
        :class="{ 'is-start': crumb.isStart, 'is-active': crumb.isActive }"
      >
        <button
          type="button"
          class="move-breadcrumb-button"
          :aria-current="crumb.isActive ? 'step' : undefined"
          :aria-label="crumb.isStart ? 'Về vị trí bắt đầu' : `Tới nước ${crumb.label}`"
          @click="jumpTo(crumb)"
        >
          <Flag v-if="crumb.isStart" :size="13" aria-hidden="true" />
          <span class="move-breadcrumb-label">{{ crumb.label }}</span>
        </button>
        <ChevronRight
          v-if="index < crumbs.length - 1"
          :size="14"
          class="move-breadcrumb-separator"
          aria-hidden="true"
        />
      </li>
    </ol>
  </nav>
</template>

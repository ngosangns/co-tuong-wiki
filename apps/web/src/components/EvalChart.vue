<script setup lang="ts">
import { Activity, Loader2 } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import type { LessonLine, LineEvaluationPly, MoveClassification } from '../api/types'
import { evaluateLine } from '../api/client'
import type { LineEvaluationPlyResult } from '../api/types'
import type { BoardState, Side } from '../core/xiangqi'
import { applyMove, initialBoard, pieceAt } from '../core/xiangqi'
import { boardFromXiangqiFen, boardToXiangqiFen } from '../engine/fen'
import { formatParsedMoveNotation } from '../composables/useMoveTrie'

const CLASSIFICATION_LABELS: Record<MoveClassification, string> = {
  best: 'Nước tốt nhất',
  excellent: 'Rất tốt',
  good: 'Tốt',
  inaccuracy: 'Chưa chính xác',
  mistake: 'Sai lầm',
  blunder: 'Sai nghiêm trọng',
  unknown: 'Chưa rõ',
}

const CLASSIFICATION_COLORS: Record<MoveClassification, string> = {
  best: 'var(--color-accent)',
  excellent: 'var(--color-primary)',
  good: 'color-mix(in srgb, var(--color-primary) 64%, transparent)',
  inaccuracy: 'color-mix(in srgb, var(--color-accent) 60%, var(--color-text-soft))',
  mistake: 'var(--color-accent)',
  blunder: 'var(--color-accent)',
  unknown: 'var(--color-text-soft)',
}

function classificationLabel(cls: MoveClassification | undefined): string {
  if (!cls) return ''
  return CLASSIFICATION_LABELS[cls] ?? ''
}

function classificationColor(cls: MoveClassification | undefined): string {
  if (!cls) return CLASSIFICATION_COLORS.unknown
  return CLASSIFICATION_COLORS[cls] ?? CLASSIFICATION_COLORS.unknown
}

const props = defineProps<{
  lines: LessonLine[]
  activeLineId: string
  activeMoveIndex: number
  initialFen?: string
}>()

const emit = defineEmits<{
  selectMove: [lineId: string, index: number]
}>()

const isLoading = ref(false)
const errorMessage = ref('')
const results = ref<LineEvaluationPlyResult[]>([])

const activeLine = computed(() => props.lines.find((line) => line.id === props.activeLineId))

function startingBoard(line: LessonLine): BoardState {
  const fen = line.initialFen ?? props.initialFen
  return fen ? boardFromXiangqiFen(fen) : initialBoard
}

function buildPliesForLine(line: LessonLine): LineEvaluationPly[] {
  const plies: LineEvaluationPly[] = []
  let board = startingBoard(line)
  const moves = line.moves ?? []
  plies.push({
    ply: 0,
    fen: boardToXiangqiFen(board, moves[0]?.side ?? 'red', 1),
    sideToMove: moves[0]?.side ?? 'red',
  })
  for (let i = 0; i < moves.length; i++) {
    const move = moves[i]
    let nextBoard = board
    try {
      nextBoard = applyMove(board, move)
    } catch {
      nextBoard = board
    }
    const nextSide: Side = move.side === 'red' ? 'black' : 'red'
    plies.push({
      ply: i + 1,
      fen: boardToXiangqiFen(nextBoard, nextSide, Math.floor((i + 1) / 2) + 1),
      sideToMove: nextSide,
      nextMove: move,
    })
    board = nextBoard
  }
  return plies
}

async function loadEvaluation() {
  const line = activeLine.value
  if (!line || !line.moves || line.moves.length === 0) {
    results.value = []
    return
  }
  isLoading.value = true
  errorMessage.value = ''
  try {
    const plies = buildPliesForLine(line)
    const openingFen = line.initialFen ?? props.initialFen
    const response = await evaluateLine({
      openingFen,
      plies,
    })
    results.value = response.plies
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Không thể đánh giá ván cờ.'
    results.value = []
  } finally {
    isLoading.value = false
  }
}

watch(
  () => [props.activeLineId, props.initialFen, props.lines] as const,
  () => {
    void loadEvaluation()
  },
  { immediate: true, deep: false },
)

const CHART_WIDTH = 640
const CHART_HEIGHT = 140
const PADDING = 24

const xScale = computed(() => {
  const count = results.value.length
  if (count <= 1) return (_: number) => CHART_WIDTH / 2
  return (ply: number) => PADDING + (ply / (count - 1)) * (CHART_WIDTH - PADDING * 2)
})

const yScale = computed(() => {
  const range = 900
  return (cp: number) => {
    const clamped = Math.max(-range, Math.min(range, cp))
    return CHART_HEIGHT / 2 - (clamped / range) * (CHART_HEIGHT / 2 - 8)
  }
})

const linePath = computed(() => {
  if (results.value.length === 0) return ''
  return results.value
    .map((result, index) => {
      const x = xScale.value(result.ply)
      const y = yScale.value(result.cp ?? 0)
      return `${index === 0 ? 'M' : 'L'}${x},${y}`
    })
    .join(' ')
})

const fillPath = computed(() => {
  if (results.value.length === 0) return ''
  const top = results.value
    .map((result, index) => `${index === 0 ? 'M' : 'L'}${xScale.value(result.ply)},${yScale.value(result.cp ?? 0)}`)
    .join(' ')
  const last = results.value[results.value.length - 1]
  const first = results.value[0]
  return `${top} L${xScale.value(last.ply)},${CHART_HEIGHT / 2} L${xScale.value(first.ply)},${CHART_HEIGHT / 2} Z`
})

const activePlyCp = computed(() => {
  const result = results.value[props.activeMoveIndex]
  if (!result) return 0
  return result.cp
})

const activePlyResult = computed(() => results.value[props.activeMoveIndex])

const activeScoreLabel = computed(() => {
  const cp = activePlyCp.value
  if (Math.abs(cp) < 60) return 'Cân bằng'
  return cp > 0 ? `Đỏ +${Math.round(cp)}` : `Đen +${Math.abs(Math.round(cp))}`
})

const activeClassification = computed<{ label: string; color: string; cpLoss?: number }>(() => {
  const result = activePlyResult.value
  if (!result?.classification) return { label: '', color: '' }
  return {
    label: classificationLabel(result.classification),
    color: classificationColor(result.classification),
    cpLoss: result.cpLoss,
  }
})

const activeBestMoveLabel = computed(() => {
  const result = results.value[props.activeMoveIndex]
  if (!result?.bestMove) return ''
  const ply = buildPliesForLine(activeLine.value!)[props.activeMoveIndex]
  const board = ply
    ? (() => {
        let b = startingBoard(activeLine.value!)
        for (let i = 0; i < props.activeMoveIndex; i++) {
          b = applyMove(b, activeLine.value!.moves![i])
        }
        return b
      })()
    : null
  const from = parseSquare(result.bestMove.slice(0, 2))
  const to = parseSquare(result.bestMove.slice(2, 4))
  if (!from || !to) return result.bestMove
  const piece = board ? pieceAt(board, from) : null
  if (!piece) return result.bestMove
  return formatParsedMoveNotation({ side: ply.sideToMove, from, to, id: '', comment: '' } as never, board)
})

function parseSquare(square: string) {
  const files = 'abcdefghi'
  const file = files.indexOf(square[0] ?? '')
  const rank = Number(square[1] ?? '0')
  if (file < 0 || Number.isNaN(rank)) return null
  return { file, rank: 9 - rank }
}

function selectPly(ply: number) {
  if (!activeLine.value) return
  emit('selectMove', activeLine.value.id, ply)
}

const cpPercent = computed(() => {
  const cp = activePlyCp.value
  const clamped = Math.max(-900, Math.min(900, cp))
  return `${50 + (clamped / 900) * 50}%`
})

const hasResults = computed(() => results.value.length > 0)
</script>

<template>
  <section class="eval-chart" aria-label="Biểu đồ đánh giá ván cờ">
    <header class="eval-chart-header">
      <div class="eval-chart-title">
        <Activity :size="16" aria-hidden="true" />
        <div>
          <h4>Đánh giá ván cờ</h4>
          <p v-if="activeBestMoveLabel">
            Ply {{ activeMoveIndex }}: <strong>{{ activeScoreLabel }}</strong> · Best: {{ activeBestMoveLabel }}
            <span v-if="activeClassification.label" class="eval-chart-classification" :style="{ color: activeClassification.color }">
              · {{ activeClassification.label
              }}<span v-if="activeClassification.cpLoss !== undefined"> (−{{ activeClassification.cpLoss }}cp)</span>
            </span>
          </p>
          <p v-else-if="hasResults">
            Ply {{ activeMoveIndex }}: <strong>{{ activeScoreLabel }}</strong>
            <span v-if="activeClassification.label" class="eval-chart-classification" :style="{ color: activeClassification.color }">
              · {{ activeClassification.label
              }}<span v-if="activeClassification.cpLoss !== undefined"> (−{{ activeClassification.cpLoss }}cp)</span>
            </span>
          </p>
          <p v-else>Đang chờ dữ liệu phân tích</p>
        </div>
      </div>
      <span v-if="isLoading" class="eval-chart-status" aria-live="polite">
        <Loader2 :size="14" class="animate-spin" aria-hidden="true" />
        Đang phân tích
      </span>
    </header>

    <div class="eval-chart-canvas">
      <svg
        v-if="hasResults"
        :viewBox="`0 0 ${CHART_WIDTH} ${CHART_HEIGHT}`"
        preserveAspectRatio="none"
        class="eval-chart-svg"
        role="img"
        aria-label="Biểu đồ đường điểm đánh giá theo nước đi"
      >
        <line
          :x1="PADDING"
          :x2="CHART_WIDTH - PADDING"
          :y1="CHART_HEIGHT / 2"
          :y2="CHART_HEIGHT / 2"
          stroke="color-mix(in srgb, var(--color-border) 70%, transparent)"
          stroke-width="1"
          stroke-dasharray="2 4"
        />
        <path :d="fillPath" fill="var(--color-accent-soft)" opacity="0.4" />
        <path :d="linePath" stroke="var(--color-accent)" stroke-width="2" fill="none" stroke-linecap="round" />
        <g class="eval-chart-points">
          <g
            v-for="result in results"
            :key="result.ply"
            :transform="`translate(${xScale(result.ply)}, ${yScale(result.cp ?? 0)})`"
            class="eval-chart-point"
            :class="{
              'is-active': result.ply === activeMoveIndex,
              'is-past': result.ply < activeMoveIndex,
              [`is-${result.classification ?? 'unknown'}`]: Boolean(result.classification),
            }"
            :title="result.classification ? `${classificationLabel(result.classification)} (−${result.cpLoss ?? 0}cp)` : ''"
          >
            <circle
              :r="result.ply === activeMoveIndex ? 6 : 3"
              :fill="classificationColor(result.classification)"
              :stroke="result.ply === activeMoveIndex ? 'var(--color-accent)' : 'transparent'"
              stroke-width="1.2"
              @click="selectPly(result.ply)"
              style="cursor: pointer"
            />
          </g>
        </g>
      </svg>
      <div v-else-if="isLoading" class="eval-chart-placeholder">
        <Loader2 :size="22" class="animate-spin" aria-hidden="true" />
        <span>Đang gọi engine cho {{ activeLine?.moves?.length ?? 0 }} nước…</span>
      </div>
      <div v-else-if="errorMessage" class="eval-chart-placeholder error">
        {{ errorMessage }}
      </div>
      <div v-else class="eval-chart-placeholder">Chưa có dữ liệu</div>
    </div>

    <footer v-if="hasResults" class="eval-chart-axis">
      <span>0</span>
      <span :style="{ left: cpPercent }" class="eval-chart-score-marker">
        <span class="eval-chart-score-marker-dot"></span>
        <span class="eval-chart-score-marker-label">{{ activeScoreLabel }}</span>
      </span>
      <span>{{ results.length - 1 }}</span>
    </footer>
  </section>
</template>

<style scoped>
.animate-spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>

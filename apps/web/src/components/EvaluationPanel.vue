<script setup lang="ts">
import { Activity, Gauge } from '@lucide/vue'
import { computed } from 'vue'
import { engineNotationToMove } from '../engine/fen'
import { pieceAt } from '../core/xiangqi'
import type { BoardState, Coordinate, LessonMove, Piece, PieceKind } from '../core/xiangqi'
import type { EngineEvaluation, EngineMove, EngineStatus } from '../engine/types'
import Card from './ui/card.vue'
import CardHeader from './ui/card-header.vue'
import CardTitle from './ui/card-title.vue'
import CardDescription from './ui/card-description.vue'
import CardContent from './ui/card-content.vue'
import Badge from './ui/badge.vue'

const props = defineProps<{
  status: EngineStatus
  board: BoardState
  evaluation: EngineEvaluation | null
  nextMove?: LessonMove
  errorMessage?: string
  compact?: boolean
}>()

const pieceNames: Record<PieceKind, string> = {
  general: 'Tướng',
  advisor: 'Sĩ',
  elephant: 'Tượng',
  horse: 'Mã',
  chariot: 'Xe',
  cannon: 'Pháo',
  soldier: 'Binh',
}

function displayFile(file: number) {
  return 9 - file
}

function displayPieceName(piece: Piece) {
  if (piece.kind === 'soldier' && piece.side === 'black') return 'Tốt'
  return pieceNames[piece.kind]
}

function usesDestinationFile(piece: Piece) {
  return piece.kind === 'horse' || piece.kind === 'elephant' || piece.kind === 'advisor'
}

function rawEngineCoordinate(square: string): Coordinate | null {
  const files = 'abcdefghi'
  const file = files.indexOf(square[0] ?? '')
  const rank = Number(square[1])

  if (file < 0 || Number.isNaN(rank) || rank < 0 || rank > 9) return null
  return { file, rank }
}

function moveCandidates(move: EngineMove) {
  const candidates: Array<{ from: Coordinate; to: Coordinate }> = [{ from: move.from, to: move.to }]
  const parsedMove = engineNotationToMove(move.notation)
  const rawFrom = rawEngineCoordinate(move.notation.slice(0, 2))
  const rawTo = rawEngineCoordinate(move.notation.slice(2, 4))

  if (parsedMove) candidates.push({ from: parsedMove.from, to: parsedMove.to })
  if (rawFrom && rawTo) candidates.push({ from: rawFrom, to: rawTo })

  return candidates
}

function formatXiangqiMove(move: EngineEvaluation['bestMove']) {
  if (!move) return 'Chưa có'

  const candidate = moveCandidates(move)
    .map((item) => ({ ...item, piece: pieceAt(props.board, item.from) }))
    .find((item) => item.piece?.side === props.evaluation?.sideToMove) ??
    moveCandidates(move)
      .map((item) => ({ ...item, piece: pieceAt(props.board, item.from) }))
      .find((item) => item.piece)

  if (!candidate?.piece) return `${move.notation.slice(0, 2)} → ${move.notation.slice(2, 4)}`

  const { from, to, piece } = candidate
  const fromFile = displayFile(from.file)
  const toFile = displayFile(to.file)
  const rankDistance = Math.abs(to.rank - from.rank)

  if (from.rank === to.rank) {
    return `${displayPieceName(piece)} ${fromFile} bình ${toFile}`
  }

  const isForward = piece.side === 'red' ? to.rank < from.rank : to.rank > from.rank
  const direction = isForward ? 'tiến' : 'thoái'
  const destination = usesDestinationFile(piece) ? toFile : rankDistance

  return `${displayPieceName(piece)} ${fromFile} ${direction} ${destination}`
}

const scoreLabel = computed(() => {
  if (!props.evaluation) return 'Chưa phân tích'

  const score = props.evaluation.score.cp
  if (Math.abs(score) < 60) return 'Cân bằng'
  return score > 0 ? `Đỏ +${Math.round(score)}` : `Đen +${Math.abs(Math.round(score))}`
})

const scoreTone = computed(() => {
  if (!props.evaluation) return 'equal'

  const score = props.evaluation?.score.cp ?? 0

  if (Math.abs(score) < 60) return 'equal'
  return score > 0 ? 'red' : 'black'
})

const scorePercent = computed(() => {
  const score = props.evaluation?.score.cp ?? 0
  const clamped = Math.max(-900, Math.min(900, score))
  return `${50 + (clamped / 900) * 50}%`
})

const bestMoveLabel = computed(() => {
  return formatXiangqiMove(props.evaluation?.bestMove)
})

const statusLabel = computed(() => {
  if (props.status === 'analyzing') return 'Đang phân tích'
  if (props.status === 'error') return 'Lỗi phân tích'
  if (props.evaluation) return props.evaluation.source === 'wukong' ? 'Wukong sẵn sàng' : 'UCI sẵn sàng'
  return 'Chờ vị trí'
})

const isAnalyzing = computed(() => props.status === 'analyzing')
const hasEvaluation = computed(() => props.status !== 'error' && !isAnalyzing.value && Boolean(props.evaluation))
</script>

<template>
  <Card
    :class="compact ? 'gap-2 p-3' : 'gap-4 p-5'"
    aria-label="Đánh giá nước đi"
    v-motion
    :initial="{ opacity: 0, scale: 0.98 }"
    :enter="{ opacity: 1, scale: 1, transition: { duration: 400, ease: 'easeOut' } }"
  >
    <CardHeader :class="compact ? 'p-0 pb-1' : 'p-0 pb-2'">
      <div class="flex items-center justify-between">
        <div>
          <CardDescription class="text-xs font-bold uppercase tracking-wider">Engine</CardDescription>
          <CardTitle :class="compact ? 'text-sm' : 'text-base'">{{ statusLabel }}</CardTitle>
        </div>
        <Activity :size="compact ? 16 : 19" class="text-primary" aria-hidden="true" />
      </div>
    </CardHeader>

    <CardContent :class="compact ? 'p-0' : 'p-0'" class="space-y-3">
      <div
        class="rounded-lg p-3"
        :class="scoreTone === 'equal' ? 'bg-muted' : scoreTone === 'red' ? 'bg-red-950/50' : 'bg-zinc-900/50'"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <Gauge :size="compact ? 16 : 18" class="text-primary" aria-hidden="true" />
            <span class="text-sm font-bold">{{ isAnalyzing ? 'Đang phân tích' : scoreLabel }}</span>
          </div>
          <Badge v-if="hasEvaluation" variant="outline" class="text-xs">
            {{ evaluation?.depth ?? 0 }} ply
          </Badge>
        </div>
        <div v-if="hasEvaluation" class="mt-2 h-2 overflow-hidden rounded-full bg-black/30">
          <div
            class="h-full rounded-full bg-gradient-to-r from-zinc-500 to-red-500 transition-all duration-300"
            :style="{ width: scorePercent }"
          />
        </div>
      </div>

      <div v-if="hasEvaluation" class="grid grid-cols-[1fr_84px] gap-3">
        <div class="rounded-lg border border-border bg-muted p-2.5">
          <span class="text-xs font-bold text-muted-foreground">Best move</span>
          <p class="text-base font-bold text-foreground truncate">{{ bestMoveLabel }}</p>
        </div>
        <div class="rounded-lg border border-border bg-muted p-2.5">
          <span class="text-xs font-bold text-muted-foreground">Depth</span>
          <p class="text-base font-bold text-foreground">{{ evaluation?.depth ?? 0 }}</p>
        </div>
      </div>

      <div
        v-if="status === 'error'"
        class="rounded-lg p-3 text-sm font-medium"
        :class="compact ? 'text-xs' : 'text-sm'"
        style="background: var(--color-accent-soft); color: var(--color-accent)"
      >
        {{ errorMessage }}
      </div>
    </CardContent>
  </Card>
</template>

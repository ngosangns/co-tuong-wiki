<script setup lang="ts">
import { Activity, Gauge } from '@lucide/vue'
import { computed } from 'vue'
import { engineNotationToMove } from '../engine/fen'
import { pieceAt } from '../core/xiangqi'
import type { BoardState, Coordinate, LessonMove, Piece, PieceKind } from '../core/xiangqi'
import type { EngineEvaluation, EngineMove, EngineStatus } from '../engine/types'

const props = defineProps<{
  status: EngineStatus
  board: BoardState
  evaluation: EngineEvaluation | null
  nextMove?: LessonMove
  errorMessage?: string
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
  <section class="evaluation-panel" aria-label="Đánh giá nước đi">
    <div class="evaluation-heading">
      <div>
        <p class="eyebrow">Engine</p>
        <h3>{{ statusLabel }}</h3>
      </div>
      <Activity :size="19" aria-hidden="true" />
    </div>

    <div class="score-card" :class="scoreTone">
      <div class="score-copy">
        <Gauge :size="18" aria-hidden="true" />
        <span>{{ isAnalyzing ? 'Đang phân tích' : scoreLabel }}</span>
      </div>
      <div v-if="hasEvaluation" class="score-track" aria-hidden="true">
        <span :style="{ width: scorePercent }"></span>
      </div>
    </div>

    <div v-if="hasEvaluation" class="engine-grid">
      <div>
        <span>Best move</span>
        <strong>{{ bestMoveLabel }}</strong>
      </div>
      <div>
        <span>Depth</span>
        <strong>{{ evaluation?.depth ?? 0 }}</strong>
      </div>
    </div>

    <p v-if="status === 'error'" class="engine-note danger">{{ errorMessage }}</p>
  </section>
</template>

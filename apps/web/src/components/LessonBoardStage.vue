<script setup lang="ts">
import { PhCaretLeft, PhCaretRight, PhWarningCircle, PhShieldCheck } from '@phosphor-icons/vue'
import type { Lesson, LessonChoice } from '../api/types'
import type { BoardState, LessonMove, Side } from '../core/xiangqi'
import type { EngineEvaluation, EngineStatus } from '../engine/types'
import EvaluationPanel from './EvaluationPanel.vue'
import OpeningSuggestions from './OpeningSuggestions.vue'
import XiangqiBoard from './XiangqiBoard.vue'

interface MoveEvaluationSource {
  status: { value: EngineStatus }
  evaluation: { value: EngineEvaluation | null }
  errorMessage: { value: string }
}

const props = defineProps<{
  lesson: Lesson
  board: BoardState
  currentMove?: LessonMove
  nextMovePreviews: LessonMove[]
  engineSuggestion?: LessonMove | null
  moveEvaluation: MoveEvaluationSource
  activeMoveComment: string
  activeMoveIndex: number
  canGoPrevious: boolean
  canGoNext: boolean
  shouldShowChoice: boolean
  visibleChoiceOptions: LessonChoice['options']
  choiceFeedback: LessonChoice['options'][number] | null
  openingDepth?: number
}>()

const emit = defineEmits<{
  selectPreviewMove: [move: LessonMove]
  selectEngineMove: [move: LessonMove]
  swipeLeft: []
  swipeRight: []
  previous: []
  next: []
  chooseMove: [moveId: string]
  playOpening: [
    move: { from: { file: number; rank: number }; to: { file: number; rank: number }; notation: string },
  ]
}>()

const sideToMove = computed<Side>(() => {
  const next = props.lesson.lines.flatMap((line) => line.moves ?? [])[props.activeMoveIndex]
  if (next) return next.side
  const last = props.lesson.lines.flatMap((line) => line.moves ?? [])[props.activeMoveIndex - 1]
  if (!last) return 'red'
  return last.side === 'red' ? 'black' : 'red'
})

const openingDepth = computed(() => props.openingDepth ?? 8)
</script>

<template>
  <section id="board-stage" class="board-stage" aria-label="Bàn cờ và các bước nước đi" tabindex="-1">
    <XiangqiBoard
      :board="props.board"
      :current-move="props.currentMove"
      :preview-moves="props.nextMovePreviews"
      :engine-suggestion="props.engineSuggestion ?? undefined"
      @select-preview-move="emit('selectPreviewMove', $event)"
      @select-engine-move="emit('selectEngineMove', $event)"
      @swipe-left="emit('swipeLeft')"
      @swipe-right="emit('swipeRight')"
    />

    <EvaluationPanel
      compact
      :status="props.moveEvaluation.status.value"
      :board="props.board"
      :evaluation="props.moveEvaluation.evaluation.value"
      :next-move="props.currentMove"
      :error-message="props.moveEvaluation.errorMessage.value"
    />

    <section
      v-if="props.activeMoveComment"
      class="board-move-comment"
      aria-live="polite"
      aria-label="Nhận xét nước hiện tại"
    >
      <p>{{ props.activeMoveComment }}</p>
    </section>

    <div class="board-controls" aria-label="Điều khiển nước đi">
      <button
        type="button"
        class="secondary-action"
        title="Nước trước"
        aria-label="Quay lại nước trước"
        :disabled="!props.canGoPrevious"
        @click="emit('previous')"
      >
        <PhCaretLeft :size="22" aria-hidden="true" />
        Nước trước
      </button>
      <button
        v-if="props.canGoNext && !props.shouldShowChoice"
        type="button"
        class="primary-action"
        @click="emit('next')"
      >
        <PhCaretRight :size="20" aria-hidden="true" />
        Nước kế
      </button>
    </div>

    <OpeningSuggestions
      :board="props.board"
      :side-to-move="sideToMove"
      :ply="props.activeMoveIndex"
      :opening-depth="openingDepth"
      @select-move="emit('playOpening', $event)"
    />

    <section v-if="props.shouldShowChoice" class="choice-box board-question">
      <div class="choice-title">
        <PhWarningCircle :size="18" aria-hidden="true" />
        <h3>{{ props.lesson.choice.prompt }}</h3>
      </div>
      <div class="choice-options">
        <button
          v-for="option in props.visibleChoiceOptions"
          :key="option.moveId"
          type="button"
          :class="option.verdict"
          @click="emit('chooseMove', option.moveId)"
        >
          {{ option.label }}
        </button>
      </div>
    </section>

    <section
      v-if="props.choiceFeedback"
      class="choice-feedback"
      :class="props.choiceFeedback.verdict"
      aria-live="polite"
      aria-label="Nhận xét nước đã chọn"
    >
      <div class="choice-feedback-title">
        <PhShieldCheck v-if="props.choiceFeedback.verdict === 'correct'" :size="18" aria-hidden="true" />
        <CircleAlert v-else :size="18" aria-hidden="true" />
        <h3>{{ props.choiceFeedback.verdict === 'correct' ? 'Nhận xét' : 'Cảnh báo' }}</h3>
      </div>
      <p>{{ props.choiceFeedback.feedback }}</p>
    </section>
  </section>
</template>

<script lang="ts">
import { computed } from 'vue'
</script>

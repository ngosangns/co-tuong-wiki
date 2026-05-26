<script setup lang="ts">
import { ChevronLeft, ChevronRight } from '@lucide/vue'
import { computed, onMounted, ref, watch } from 'vue'
import { fetchCombinedLesson, fetchCombinedNextSteps } from '../api/client'
import type { Lesson, LessonLine, LessonPhase } from '../api/types'
import { useLessonPlayer } from '../composables/useLessonPlayer'
import { useMoveEvaluation } from '../composables/useMoveEvaluation'
import type { LessonMove } from '../core/xiangqi'
import EvaluationPanel from './EvaluationPanel.vue'
import MoveGraph from './MoveGraph.vue'
import XiangqiBoard from './XiangqiBoard.vue'

const emptyLesson: Lesson = {
  id: '',
  title: 'Tổng hợp toàn bộ lesson',
  category: 'Tổng hợp',
  difficulty: 'Tất cả',
  lines: [],
  choice: {
    prompt: '',
    options: [],
  },
}

const lesson = ref<Lesson>(emptyLesson)
const isLoading = ref(true)
const errorMessage = ref('')
const loadingPhases = ref<Record<LessonPhase, boolean>>({ opening: false, middlegame: false, endgame: false })
const selectedPhase = ref<LessonPhase>('opening')
const player = useLessonPlayer(lesson)
const moveEvaluation = useMoveEvaluation(player)

const phases: Array<{ id: LessonPhase; title: string }> = [
  { id: 'opening', title: 'Khai cuộc' },
  { id: 'middlegame', title: 'Trung cuộc' },
  { id: 'endgame', title: 'Tàn cuộc' },
]

const phaseSections = computed(() =>
  phases
    .map((phase) => ({
      ...phase,
      lines: lesson.value.lines.filter((line) => line.phase === phase.id),
    }))
    .filter((phase) => phase.lines.length > 0),
)
const activePhase = computed(() => linePhase(player.activeLineId.value))
const selectedPhaseSection = computed(() => phaseSections.value.find((section) => section.id === selectedPhase.value) ?? phaseSections.value[0])
const activeMovePrefix = computed(() => player.activeMoves.value.slice(0, player.activeMoveIndex.value).map(moveSignature))
const activeStartKey = computed(() => {
  const activeLine = player.activeLine.value
  return activeLine ? startKeyForLine(activeLine) : ''
})
const activeNodeLines = computed(() => {
  const section = phaseSections.value.find((item) => item.id === activePhase.value)
  if (!section || !activeStartKey.value) return []

  return section.lines.filter((line) => startKeyForLine(line) === activeStartKey.value && lineMatchesActivePrefix(line))
})
const nextGraphChoices = computed(() => {
  const choices = new Set<string>()

  activeNodeLines.value.forEach((line) => {
    const nextMove = line.moves?.[player.activeMoveIndex.value]
    if (nextMove) choices.add(moveSignature(nextMove))
  })

  return choices
})
const canGoPrevious = computed(() => player.activeMoveIndex.value > 0)
const canGoNext = computed(() => !loadingPhases.value[activePhase.value] && nextGraphChoices.value.size === 1)

function updateLineMoves(lineId: string, from: number, moves: LessonMove[], totalMoves: number) {
  const line = lesson.value.lines.find((item) => item.id === lineId)
  if (!line) return

  const nextMoves = (line.moves ?? []).slice()
  moves.forEach((move, index) => {
    nextMoves[from + index] = move
  })
  line.moves = nextMoves
  line.moveCount = totalMoves
}

function linePhase(lineId: string): LessonPhase {
  return lesson.value.lines.find((line) => line.id === lineId)?.phase ?? 'opening'
}

function moveSignature(move: LessonMove) {
  return `${move.side}:${move.from.file},${move.from.rank}:${move.to.file},${move.to.rank}`
}

function startKeyForLine(line: LessonLine) {
  return (line.initialFen ?? lesson.value.initialFen ?? 'standard').trim()
}

function lineMatchesActivePrefix(line: LessonLine) {
  const moves = line.moves ?? []
  return activeMovePrefix.value.every((signature, index) => {
    const move = moves[index]
    return move ? moveSignature(move) === signature : false
  })
}

async function ensureActiveNodeMoves(count: number) {
  const phase = activePhase.value
  const lines = activeNodeLines.value
  if (!lines.length || loadingPhases.value[phase] || lines.every((line) => (line.moves?.length ?? 0) >= count)) return

  loadingPhases.value = { ...loadingPhases.value, [phase]: true }
  try {
    const window = await fetchCombinedNextSteps({
      phase,
      initialFen: activeStartKey.value,
      from: player.activeMoveIndex.value,
      limit: Math.max(1, count - player.activeMoveIndex.value),
      prefix: player.activeMoves.value.slice(0, player.activeMoveIndex.value),
    })
    window.lines.forEach((line) => updateLineMoves(line.lineId, line.from, line.moves, line.totalMoves))
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Không thể tải nước tiếp theo.'
  } finally {
    loadingPhases.value = { ...loadingPhases.value, [phase]: false }
  }
}

async function selectPhase(phase: LessonPhase) {
  selectedPhase.value = phase
  const firstLine = phaseSections.value.find((section) => section.id === phase)?.lines[0]
  if (firstLine) {
    player.setLine(firstLine.id)
    player.goToMove(0)
    await ensureActiveNodeMoves(1)
  }
}

async function selectLine(lineId: string) {
  player.setLine(lineId)
  await ensureActiveNodeMoves(player.activeMoveIndex.value + 1)
}

async function goToGraphMove(lineId: string, index: number) {
  player.setLine(lineId)
  player.goToMove(index)
  await ensureActiveNodeMoves(index + 1)
}

async function goToPreviousStep() {
  if (!canGoPrevious.value) return
  player.previous()
}

async function goToNextStep() {
  await ensureActiveNodeMoves(player.activeMoveIndex.value + 1)
  if (!canGoNext.value) return
  player.next()
}

onMounted(async () => {
  try {
    const overview = await fetchCombinedLesson()
    lesson.value = overview
    await ensureActiveNodeMoves(1)
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Không thể tải lesson tổng hợp.'
  } finally {
    isLoading.value = false
  }
})

watch(activePhase, (phase) => {
  selectedPhase.value = phase
})

watch(
  () => [activePhase.value, player.activeMoveIndex.value] as const,
  ([, moveIndex]) => {
    void ensureActiveNodeMoves(moveIndex + 1)
  },
)
</script>

<template>
  <main class="combined-page">
    <div v-if="errorMessage || isLoading" class="app-status" role="status">
      {{ errorMessage || 'Đang tải lesson tổng hợp...' }}
    </div>

    <section class="combined-board" aria-label="Bàn cờ tổng hợp">
      <div class="combined-board-stage">
        <XiangqiBoard :board="player.board.value" :current-move="player.currentMove.value" />

        <div class="board-controls combined-step-controls" aria-label="Điều khiển nước đi tổng hợp">
          <button
            type="button"
            class="secondary-action"
            title="Previous step"
            aria-label="Previous step"
            :disabled="!canGoPrevious"
            @click="goToPreviousStep"
          >
            <ChevronLeft :size="22" aria-hidden="true" />
            Previous step
          </button>
          <button
            type="button"
            class="primary-action"
            title="Next step"
            aria-label="Next step"
            :disabled="!canGoNext"
            @click="goToNextStep"
          >
            <ChevronRight :size="20" aria-hidden="true" />
            Next step
          </button>
        </div>
      </div>
    </section>

    <section class="combined-graph" aria-label="Cây nước đi tổng hợp">
      <MoveGraph
        v-if="selectedPhaseSection"
        :title="selectedPhaseSection.title"
        :lines="selectedPhaseSection.lines"
        :active-line-id="player.activeLineId.value"
        :active-move-index="player.activeMoveIndex.value"
        :initial-fen="lesson.initialFen"
        @select-line="selectLine"
        @select-move="goToGraphMove"
      >
        <template #toolbar="{ fit }">
          <div class="combined-graph-tabs" role="tablist" aria-label="Giai đoạn ván cờ">
            <button
              v-for="section in phaseSections"
              :key="section.id"
              type="button"
              class="combined-graph-tab"
              :class="{ active: selectedPhase === section.id, current: activePhase === section.id }"
              role="tab"
              :aria-selected="selectedPhase === section.id"
              @click="selectPhase(section.id)"
            >
              <span>{{ section.title }}</span>
              <small>{{ section.lines.length }} biến</small>
            </button>
          </div>

          <button type="button" class="graph-fit-button" title="Canh giữa graph" aria-label="Canh giữa graph" @click="fit">
            Fit
          </button>
        </template>
      </MoveGraph>
    </section>

    <section class="combined-engine" aria-label="Engine">
      <EvaluationPanel
        :status="moveEvaluation.status.value"
        :board="player.board.value"
        :evaluation="moveEvaluation.evaluation.value"
        :next-move="moveEvaluation.nextMove.value"
        :error-message="moveEvaluation.errorMessage.value"
      />
    </section>
  </main>
</template>

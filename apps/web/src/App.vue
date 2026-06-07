<script setup lang="ts">
import { ref } from 'vue'
import CombinedLessonPage from './components/CombinedLessonPage.vue'
import EvalChart from './components/EvalChart.vue'
import LibraryPanel from './components/LibraryPanel.vue'
import LessonBoardStage from './components/LessonBoardStage.vue'
import LessonInspectorPanel from './components/LessonInspectorPanel.vue'
import LessonMobileDock from './components/LessonMobileDock.vue'
import LessonMobileTabs from './components/LessonMobileTabs.vue'
import LessonTopbar from './components/LessonTopbar.vue'
import MoveBreadcrumb from './components/MoveBreadcrumb.vue'
import MoveGraph from './components/MoveGraph.vue'
import MoveMinimap from './components/MoveMinimap.vue'
import { useLessonWorkspace } from './composables/useLessonWorkspace'
import type { LessonMove } from './core/xiangqi'

const workspace = useLessonWorkspace()
const {
  activeCategory,
  activeLessonId,
  activeMoveComment,
  activeMobileTab,
  canGoNext,
  canGoPrevious,
  categoryStats,
  choiceFeedback,
  errorMessage,
  goToGraphMove,
  goToPreviewMove,
  isCombinedPage,
  isLoading,
  isPrinciplesExpanded,
  isSidebarOpen,
  lesson,
  mobileTabs,
  moveEvaluation,
  nextMovePreviews,
  player,
  principleCount,
  selectLessonAndResetTab,
  shouldShowChoice,
  summaries,
  toggleCategory,
  togglePrinciples,
  toggleSidebar,
  visibleChoiceOptions,
} = workspace

const showEngineSuggestion = ref(false)
const engineBestMove = ref<LessonMove | null>(null)

function onEngineBestMove(
  notation: {
    from: { file: number; rank: number }
    to: { file: number; rank: number }
    side: 'red' | 'black'
    uci: string
  } | null,
) {
  if (!notation) {
    engineBestMove.value = null
    return
  }
  engineBestMove.value = {
    id: `engine-${notation.uci}`,
    side: notation.side,
    from: notation.from,
    to: notation.to,
    comment: 'Engine suggestion (so sánh)',
  }
}

function onSelectEngineMove() {
  const suggestion = engineBestMove.value
  if (!suggestion) return
  for (const line of lesson.value.lines) {
    const moves = line.moves ?? []
    const next = moves[player.activeMoveIndex.value]
    if (
      next &&
      next.from.file === suggestion.from.file &&
      next.from.rank === suggestion.from.rank &&
      next.to.file === suggestion.to.file &&
      next.to.rank === suggestion.to.rank
    ) {
      player.setLine(line.id)
      player.goToMove(player.activeMoveIndex.value + 1)
      return
    }
  }
  goToPreviewMove(suggestion)
}

function openGraph() {
  activeMobileTab.value = 'graph'
}

function playOpeningMove(notation: {
  from: { file: number; rank: number }
  to: { file: number; rank: number }
  notation: string
}) {
  // Walk any line whose next ply matches the move coordinates.
  for (const line of lesson.value.lines) {
    const moves = line.moves ?? []
    const next = moves[player.activeMoveIndex.value]
    if (
      next &&
      next.from.file === notation.from.file &&
      next.from.rank === notation.from.rank &&
      next.to.file === notation.to.file &&
      next.to.rank === notation.to.rank
    ) {
      player.setLine(line.id)
      player.goToMove(player.activeMoveIndex.value + 1)
      return
    }
  }
  // Fallback: synthesize a LessonMove from the suggestion and re-use the
  // existing preview-jump path. This also covers cases where the lesson
  // does not include the opening move but the user wants to advance the ply.
  const syntheticMove = {
    id: `opening-${notation.from.file},${notation.from.rank}-${notation.to.file},${notation.to.rank}`,
    side: player.activeMoves.value[player.activeMoveIndex.value - 1]?.side === 'red' ? 'black' : 'red',
    from: notation.from,
    to: notation.to,
    comment: '',
  } as LessonMove
  goToPreviewMove(syntheticMove)
}
</script>

<template>
  <a class="skip-link" href="#board-stage">Bỏ qua đến bàn cờ</a>
  <CombinedLessonPage v-if="isCombinedPage" />
  <main v-else>
    <div v-if="errorMessage || isLoading" class="app-status" role="status">
      {{ errorMessage || 'Đang tải dữ liệu bài học...' }}
    </div>

    <div class="app-shell">
      <div v-if="isSidebarOpen" class="sidebar-overlay" @click="isSidebarOpen = false"></div>

      <LibraryPanel
        :is-open="isSidebarOpen"
        :lesson-title="lesson.title"
        :category-stats="categoryStats"
        :active-category="activeCategory"
        :active-lesson-id="activeLessonId"
        :summaries="summaries"
        :is-category-expanded="(category) => workspace.isCategoryExpanded(category)"
        @toggle="toggleSidebar"
        @close="isSidebarOpen = false"
        @select-lesson="selectLessonAndResetTab"
        @toggle-category="toggleCategory"
      />

      <LessonTopbar :title="lesson.title" :category="lesson.category" :difficulty="lesson.difficulty" />

      <section class="breadcrumb-row" aria-label="Đường đi nước cờ hiện tại">
        <MoveBreadcrumb
          :lines="lesson.lines"
          :active-line-id="player.activeLineId.value"
          :active-move-index="player.activeMoveIndex.value"
          :initial-fen="lesson.initialFen"
          @select-move="goToGraphMove"
        />
      </section>

      <section class="board-panel" aria-label="Bàn học">
        <LessonBoardStage
          v-show="activeMobileTab === 'board'"
          :lesson="lesson"
          :board="player.board.value"
          :current-move="player.currentMove.value"
          :next-move-previews="nextMovePreviews"
          :engine-suggestion="engineBestMove"
          :move-evaluation="moveEvaluation"
          :active-move-comment="activeMoveComment"
          :active-move-index="player.activeMoveIndex.value"
          :can-go-previous="canGoPrevious"
          :can-go-next="canGoNext"
          :should-show-choice="shouldShowChoice"
          :visible-choice-options="visibleChoiceOptions"
          :choice-feedback="choiceFeedback"
          @select-preview-move="goToPreviewMove"
          @select-engine-move="onSelectEngineMove"
          @swipe-left="player.next"
          @swipe-right="player.previous"
          @previous="player.previous"
          @next="player.next"
          @choose-move="player.chooseMove"
          @play-opening="playOpeningMove"
        />

        <section class="board-side-panel desktop-only" aria-label="Điều khiển và phản hồi bài học">
          <MoveMinimap
            :lines="lesson.lines"
            :active-line-id="player.activeLineId.value"
            :active-move-index="player.activeMoveIndex.value"
            :initial-fen="lesson.initialFen"
            @select-move="goToGraphMove"
            @open-graph="openGraph"
          />
          <MoveGraph
            :lines="lesson.lines"
            :active-line-id="player.activeLineId.value"
            :active-move-index="player.activeMoveIndex.value"
            :initial-fen="lesson.initialFen"
            @select-line="player.setLine"
            @select-move="goToGraphMove"
          />
        </section>

        <LessonMobileTabs
          v-model="activeMobileTab"
          :lesson="lesson"
          :active-line-id="player.activeLineId.value"
          :active-move-index="player.activeMoveIndex.value"
          :is-principles-expanded="isPrinciplesExpanded"
          :principle-count="principleCount"
          :tabs="mobileTabs"
          @select-line="player.setLine"
          @select-move="goToGraphMove"
          @toggle-principles="togglePrinciples"
        />

        <EvalChart
          v-if="lesson.lines.length > 0"
          class="eval-chart-row"
          :lines="lesson.lines"
          :active-line-id="player.activeLineId.value"
          :active-move-index="player.activeMoveIndex.value"
          :initial-fen="lesson.initialFen"
          :show-engine-suggestion="showEngineSuggestion"
          @update:show-engine-suggestion="showEngineSuggestion = $event"
          @engine-best-move="onEngineBestMove"
          @select-move="goToGraphMove"
        />
      </section>

      <section class="lesson-panel desktop-only" aria-label="Nội dung bài học">
        <LessonInspectorPanel
          :is-expanded="isPrinciplesExpanded"
          :principle-count="principleCount"
          @toggle="togglePrinciples"
        />
      </section>

      <LessonMobileDock
        class="mobile-only"
        :active-tab="activeMobileTab"
        @update:active-tab="activeMobileTab = $event"
      />
    </div>
  </main>
</template>

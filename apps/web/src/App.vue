<script setup lang="ts">
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

function openGraph() {
  activeMobileTab.value = 'graph'
}
</script>

<template>
  <CombinedLessonPage v-if="isCombinedPage" />
  <main v-else>
    <div v-if="errorMessage || isLoading" class="app-status" role="status">
      {{ errorMessage || 'Đang tải dữ liệu bài học...' }}
    </div>

    <div
      class="app-shell"
      v-motion
      :initial="{ opacity: 0, y: 8 }"
      :enter="{ opacity: 1, y: 0, transition: { duration: 500, ease: 'easeOut' } }"
    >
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

      <LessonTopbar
        :title="lesson.title"
        :category="lesson.category"
        :difficulty="lesson.difficulty"
      />

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
          :move-evaluation="moveEvaluation"
          :active-move-comment="activeMoveComment"
          :can-go-previous="canGoPrevious"
          :can-go-next="canGoNext"
          :should-show-choice="shouldShowChoice"
          :visible-choice-options="visibleChoiceOptions"
          :choice-feedback="choiceFeedback"
          @select-preview-move="goToPreviewMove"
          @swipe-left="player.next"
          @swipe-right="player.previous"
          @previous="player.previous"
          @next="player.next"
          @choose-move="player.chooseMove"
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

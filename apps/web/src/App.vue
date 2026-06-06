<script setup lang="ts">
import { BookOpen, ChevronDown, ChevronLeft, ChevronRight, CircleAlert, GitBranch, Lightbulb, Menu, Search, ShieldCheck, X } from '@lucide/vue'
import { computed, onMounted, ref } from 'vue'
import CombinedLessonPage from './components/CombinedLessonPage.vue'
import EvaluationPanel from './components/EvaluationPanel.vue'
import MoveGraph from './components/MoveGraph.vue'
import XiangqiBoard from './components/XiangqiBoard.vue'
import { useLessonPlayer } from './composables/useLessonPlayer'
import { useLessons } from './composables/useLessons'
import { useMoveEvaluation } from './composables/useMoveEvaluation'
import { lessonPrincipleGroups } from './content/principles'
import { moveSignature, nextMoveTargetsForActiveNode } from './core/movePreview'
import type { LessonMove } from './core/xiangqi'
import type { Lesson } from './api/types'

const emptyLesson: Lesson = {
  id: '',
  title: 'Đang tải bài học',
  category: '',
  difficulty: '',
  initialFen: undefined,
  lines: [],
  choice: {
    prompt: '',
    options: [],
  },
}

const {
  activeCategory,
  activeLesson,
  activeLessonId,
  categoryStats,
  errorMessage,
  isCategoryExpanded,
  isLoading,
  loadCatalog,
  selectLesson,
  summaries,
  toggleCategory,
} = useLessons()
const lesson = computed(() => activeLesson.value ?? emptyLesson)
const player = useLessonPlayer(lesson)
const moveEvaluation = useMoveEvaluation(player)
const isPrinciplesExpanded = ref(true)
const isSidebarOpen = ref(false)
const isCombinedPage = ref(window.location.pathname.replace(/\/$/, '') === '/combined')
const principleCount = computed(() => lessonPrincipleGroups.reduce((count, group) => count + group.items.length, 0))

const choicePromptPly = computed(() => {
  const choiceMoveIds = new Set(lesson.value.choice.options.map((option) => option.moveId))
  const optionIndexes = lesson.value.lines
    .flatMap((line) => (line.moves ?? []).map((move, index) => (choiceMoveIds.has(move.id) ? index : -1)))
    .filter((index) => index >= 0)

  return optionIndexes.length ? Math.min(...optionIndexes) : -1
})

const visibleChoiceOptions = computed(() => {
  if (choicePromptPly.value < 0) return []

  const moveIdsAtPrompt = new Set(
    lesson.value.lines.map((line) => line.moves?.[choicePromptPly.value]?.id).filter((moveId): moveId is string => Boolean(moveId)),
  )

  return lesson.value.choice.options.filter((option) => moveIdsAtPrompt.has(option.moveId))
})
const shouldShowChoice = computed(
  () => lesson.value.choice.prompt && player.activeMoveIndex.value === choicePromptPly.value && visibleChoiceOptions.value.length > 1,
)
const choiceFeedback = computed(() => player.selectedChoice.value)
const activeMoveComment = computed(() => player.currentMove.value?.comment ?? '')
const nextMoveTargets = computed(() =>
  nextMoveTargetsForActiveNode({
    lines: lesson.value.lines,
    activeLine: player.activeLine.value,
    activeMoveIndex: player.activeMoveIndex.value,
    lessonInitialFen: lesson.value.initialFen,
  }),
)
const nextMovePreviews = computed(() => nextMoveTargets.value.map((target) => target.move))
const canGoPrevious = computed(() => player.activeMoveIndex.value > 0)
const canGoNext = computed(() => player.activeMoveIndex.value < player.activeMoves.value.length)

function goToGraphMove(lineId: string, index: number) {
  player.setLine(lineId)
  player.goToMove(index)
}

function goToPreviewMove(move: LessonMove) {
  const target = nextMoveTargets.value.find((item) => moveSignature(item.move) === moveSignature(move))
  if (!target) return

  goToGraphMove(target.lineId, player.activeMoveIndex.value + 1)
}

function lessonsForCategory(category: string) {
  return summaries.value.filter((item) => item.category === category)
}

function toggleSidebar() {
  isSidebarOpen.value = !isSidebarOpen.value
}

onMounted(() => {
  if (!isCombinedPage.value) {
    loadCatalog()
  }
})
</script>

<template>
  <CombinedLessonPage v-if="isCombinedPage" />
  <main v-else>
    <div v-if="errorMessage || isLoading" class="app-status" role="status">
      {{ errorMessage || 'Đang tải dữ liệu bài học...' }}
    </div>

    <div class="app-shell">
      <button
        type="button"
        class="mobile-nav-toggle"
        aria-label="Mở danh sách bài học"
        :aria-expanded="isSidebarOpen"
        @click="toggleSidebar"
      >
        <Menu v-if="!isSidebarOpen" :size="20" aria-hidden="true" />
        <X v-else :size="20" aria-hidden="true" />
        <span>Danh mục</span>
      </button>

      <aside class="library-panel" :class="{ 'is-open': isSidebarOpen }">
        <div class="brand-lockup">
          <div class="brand-mark">象</div>
          <div>
            <p class="eyebrow">Cờ Tướng Wiki</p>
            <h1>Học bằng thế cờ thật</h1>
          </div>
        </div>

        <label class="search-box">
          <Search :size="18" aria-hidden="true" />
          <input type="search" placeholder="Tìm khai cuộc, cạm bẫy..." />
        </label>

        <a class="combined-nav-link" href="/combined">
          <GitBranch :size="18" aria-hidden="true" />
          <span>Tổng hợp toàn bộ lesson</span>
        </a>

        <section class="topic-list" aria-label="Chủ đề và bài học">
          <div v-for="category in categoryStats" :key="category.name" class="topic-group">
            <button
              class="topic"
              :class="{ active: activeCategory === category.name, expanded: isCategoryExpanded(category.name) }"
              type="button"
              :aria-expanded="isCategoryExpanded(category.name)"
              @click="toggleCategory(category.name)"
            >
              <BookOpen :size="18" aria-hidden="true" />
              <span>{{ category.name }}</span>
              <small>{{ category.count }}</small>
            </button>

            <div v-if="isCategoryExpanded(category.name)" class="lesson-child-list">
              <button
                v-for="(item, lessonIndex) in lessonsForCategory(category.name)"
                :key="item.id"
                type="button"
                class="lesson-child"
                :class="{ active: activeLessonId === item.id }"
                @click="selectLesson(item.id); isSidebarOpen = false"
              >
                <small class="lesson-child-index">{{ lessonIndex + 1 }}</small>
                <span>{{ item.title }}</span>
              </button>
            </div>
          </div>
        </section>
      </aside>

      <section class="lesson-topbar" aria-label="Bài học">
        <article class="wiki-article">
          <p class="eyebrow">{{ lesson.category }} · {{ lesson.difficulty }}</p>
          <h2>{{ lesson.title }}</h2>
        </article>

      </section>

      <section class="board-panel" aria-label="Bàn học">
        <section class="board-stage" aria-label="Bàn cờ và các bước nước đi">
          <XiangqiBoard
            :board="player.board.value"
            :current-move="player.currentMove.value"
            :preview-moves="nextMovePreviews"
            @select-preview-move="goToPreviewMove"
          />

          <EvaluationPanel
            compact
            :status="moveEvaluation.status.value"
            :board="player.board.value"
            :evaluation="moveEvaluation.evaluation.value"
            :next-move="moveEvaluation.nextMove.value"
            :error-message="moveEvaluation.errorMessage.value"
          />

          <section v-if="activeMoveComment" class="board-move-comment" aria-live="polite" aria-label="Nhận xét nước hiện tại">
            <p>{{ activeMoveComment }}</p>
          </section>

          <div class="board-controls" aria-label="Điều khiển nước đi">
            <button
              type="button"
              class="secondary-action"
              title="Nước trước"
              aria-label="Quay lại nước trước"
              :disabled="!canGoPrevious"
              @click="player.previous"
            >
              <ChevronLeft :size="22" aria-hidden="true" />
              Nước trước
            </button>
            <button v-if="canGoNext && !shouldShowChoice" type="button" class="primary-action" @click="player.next">
              <ChevronRight :size="20" aria-hidden="true" />
              Nước kế
            </button>
          </div>

          <section v-if="shouldShowChoice" class="choice-box board-question">
            <div class="choice-title">
              <CircleAlert :size="18" aria-hidden="true" />
              <h3>{{ lesson.choice.prompt }}</h3>
            </div>
            <div class="choice-options">
              <button
                v-for="option in visibleChoiceOptions"
                :key="option.moveId"
                type="button"
                :class="option.verdict"
                @click="player.chooseMove(option.moveId)"
              >
                {{ option.label }}
              </button>
            </div>
          </section>

          <section
            v-if="choiceFeedback"
            class="choice-feedback"
            :class="choiceFeedback.verdict"
            aria-live="polite"
            aria-label="Nhận xét nước đã chọn"
          >
            <div class="choice-feedback-title">
              <ShieldCheck v-if="choiceFeedback.verdict === 'correct'" :size="18" aria-hidden="true" />
              <CircleAlert v-else :size="18" aria-hidden="true" />
              <h3>{{ choiceFeedback.verdict === 'correct' ? 'Nhận xét' : 'Cảnh báo' }}</h3>
            </div>
            <p>{{ choiceFeedback.feedback }}</p>
          </section>
        </section>

        <section class="board-side-panel" aria-label="Điều khiển và phản hồi bài học">
          <MoveGraph
            :lines="lesson.lines"
            :active-line-id="player.activeLineId.value"
            :active-move-index="player.activeMoveIndex.value"
            :initial-fen="lesson.initialFen"
            @select-line="player.setLine"
            @select-move="goToGraphMove"
          />
        </section>
      </section>

      <section class="lesson-panel" aria-label="Nội dung bài học">
        <section class="principles" :class="{ expanded: isPrinciplesExpanded }">
          <button
            type="button"
            class="principles-toggle"
            :aria-expanded="isPrinciplesExpanded"
            aria-controls="lesson-principles"
            @click="isPrinciplesExpanded = !isPrinciplesExpanded"
          >
            <span class="principles-title">
              <Lightbulb :size="18" aria-hidden="true" />
              <span>
                <span class="principles-eyebrow">Tổng hợp</span>
                <strong>Điểm cần nhớ</strong>
              </span>
            </span>
            <span class="principles-meta">
              {{ principleCount }} ý
              <ChevronDown :size="18" aria-hidden="true" />
            </span>
          </button>

          <div v-show="isPrinciplesExpanded" id="lesson-principles" class="principles-groups">
            <section v-for="group in lessonPrincipleGroups" :key="group.id" class="principle-group">
              <header>
                <h3>{{ group.title }}</h3>
                <span>{{ group.items.length }}</span>
              </header>
              <ul class="principles-list">
                <li v-for="principle in group.items" :key="principle">{{ principle }}</li>
              </ul>
            </section>
          </div>
        </section>
      </section>
    </div>
  </main>
</template>

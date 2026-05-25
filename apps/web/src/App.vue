<script setup lang="ts">
import { BookOpen, ChevronLeft, ChevronRight, CircleAlert, Monitor, Moon, Search, ShieldCheck, Sun } from '@lucide/vue'
import { computed, onMounted, ref, watch } from 'vue'
import EvaluationPanel from './components/EvaluationPanel.vue'
import MoveGraph from './components/MoveGraph.vue'
import XiangqiBoard from './components/XiangqiBoard.vue'
import { useLessonPlayer } from './composables/useLessonPlayer'
import { useLessons } from './composables/useLessons'
import { useMoveEvaluation } from './composables/useMoveEvaluation'
import type { Lesson } from './api/types'

type ThemeMode = 'light' | 'dark' | 'system'

const emptyLesson: Lesson = {
  id: '',
  slug: '',
  title: 'Đang tải bài học',
  summary: '',
  category: '',
  difficulty: '',
  tags: [],
  principles: [],
  initialFen: undefined,
  lines: [],
  choice: {
    id: '',
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
  isLoading,
  loadCatalog,
  selectCategory,
  selectLesson,
  summaries,
} = useLessons()
const lesson = computed(() => activeLesson.value ?? emptyLesson)
const player = useLessonPlayer(lesson)
const moveEvaluation = useMoveEvaluation(player)
const themeMode = ref<ThemeMode>('system')

const themeOptions: Array<{
  value: ThemeMode
  label: string
  icon: typeof Sun
}> = [
  { value: 'light', label: 'Sáng', icon: Sun },
  { value: 'dark', label: 'Tối', icon: Moon },
  { value: 'system', label: 'Máy', icon: Monitor },
]

const activeThemeOption = computed(() => themeOptions.find((option) => option.value === themeMode.value) ?? themeOptions[2])

function applyTheme(mode: ThemeMode) {
  document.documentElement.dataset.theme = mode
}

function toggleTheme() {
  const currentIndex = themeOptions.findIndex((option) => option.value === themeMode.value)
  themeMode.value = themeOptions[(currentIndex + 1) % themeOptions.length].value
}

const choicePromptPly = computed(() => {
  const choiceMoveIds = new Set(lesson.value.choice.options.map((option) => option.moveId))
  const optionIndexes = lesson.value.lines
    .flatMap((line) => line.moves.map((move, index) => (choiceMoveIds.has(move.id) ? index : -1)))
    .filter((index) => index >= 0)

  return optionIndexes.length ? Math.min(...optionIndexes) : -1
})

const shouldShowChoice = computed(() => lesson.value.choice.prompt && player.activeMoveIndex.value === choicePromptPly.value)
const choiceFeedback = computed(() => player.selectedChoice.value)
const canGoPrevious = computed(() => player.activeMoveIndex.value > 0)
const canGoNext = computed(() => player.activeMoveIndex.value < player.activeMoves.value.length)

function goToGraphMove(lineId: string, index: number) {
  player.setLine(lineId)
  player.goToMove(index)
}

function lessonsForCategory(category: string) {
  return summaries.value.filter((item) => item.category === category)
}

onMounted(() => {
  loadCatalog()

  const savedTheme = localStorage.getItem('co-tuong-theme') as ThemeMode | null

  if (savedTheme === 'light' || savedTheme === 'dark' || savedTheme === 'system') {
    themeMode.value = savedTheme
  }

  applyTheme(themeMode.value)
})

watch(themeMode, (mode) => {
  localStorage.setItem('co-tuong-theme', mode)
  applyTheme(mode)
})
</script>

<template>
  <main>
    <div v-if="errorMessage || isLoading" class="app-status" role="status">
      {{ errorMessage || 'Đang tải dữ liệu bài học...' }}
    </div>

    <div class="app-shell">
      <aside class="library-panel">
        <div class="brand-lockup">
          <div class="brand-mark">象</div>
          <div>
            <p class="eyebrow">Cờ Tướng Wiki</p>
            <h1>Học bằng thế cờ thật</h1>
          </div>
        </div>

        <button
          type="button"
          class="theme-toggle"
          :title="`Giao diện ${activeThemeOption.label}`"
          :aria-label="`Giao diện hiện tại: ${activeThemeOption.label}. Bấm để đổi giao diện.`"
          @click="toggleTheme"
        >
          <component :is="activeThemeOption.icon" :size="16" aria-hidden="true" />
          <span>{{ activeThemeOption.label }}</span>
        </button>

        <label class="search-box">
          <Search :size="18" aria-hidden="true" />
          <input type="search" placeholder="Tìm khai cuộc, cạm bẫy..." />
        </label>

        <section class="topic-list" aria-label="Chủ đề và bài học">
          <div v-for="category in categoryStats" :key="category.name" class="topic-group">
            <button
              class="topic"
              :class="{ active: activeCategory === category.name }"
              type="button"
              @click="selectCategory(category.name)"
            >
              <BookOpen :size="18" aria-hidden="true" />
              <span>{{ category.name }}</span>
              <small>{{ category.count }}</small>
            </button>

            <div v-if="activeCategory === category.name" class="lesson-child-list">
              <button
                v-for="item in lessonsForCategory(category.name)"
                :key="item.id"
                type="button"
                class="lesson-child"
                :class="{ active: activeLessonId === item.id }"
                @click="selectLesson(item.id)"
              >
                <span>{{ item.title }}</span>
                <small>{{ item.difficulty }}</small>
              </button>
            </div>
          </div>
        </section>
      </aside>

      <section class="lesson-topbar" aria-label="Tóm tắt bài học">
        <article class="wiki-article">
          <p class="eyebrow">{{ lesson.category }} · {{ lesson.difficulty }}</p>
          <h2>{{ lesson.title }}</h2>
          <p>{{ lesson.summary }}</p>
          <div class="tag-row">
            <span v-for="tag in lesson.tags" :key="tag">{{ tag }}</span>
          </div>
        </article>

        <section class="principles">
          <h3>Điểm cần nhớ</h3>
          <ul>
            <li v-for="principle in lesson.principles" :key="principle">{{ principle }}</li>
          </ul>
        </section>
      </section>

      <section class="board-panel" aria-label="Bàn học">
        <section v-if="shouldShowChoice" class="choice-box board-question">
          <div class="choice-title">
            <CircleAlert :size="18" aria-hidden="true" />
            <h3>{{ lesson.choice.prompt }}</h3>
          </div>
          <div class="choice-options">
            <button
              v-for="option in lesson.choice.options"
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

        <section class="board-stage" aria-label="Bàn cờ và các bước nước đi">
          <XiangqiBoard :board="player.board.value" :current-move="player.currentMove.value" />
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
          <button v-if="canGoNext" type="button" class="primary-action" @click="player.next">
            <ChevronRight :size="20" aria-hidden="true" />
            Nước kế
          </button>
        </div>
      </section>

      <section class="lesson-panel" aria-label="Nội dung bài học">
        <MoveGraph
          :lines="lesson.lines"
          :active-line-id="player.activeLineId.value"
          :active-move-index="player.activeMoveIndex.value"
          @select-line="player.setLine"
          @select-move="goToGraphMove"
        />

        <EvaluationPanel
          :status="moveEvaluation.status.value"
          :board="player.board.value"
          :evaluation="moveEvaluation.evaluation.value"
          :next-move="moveEvaluation.nextMove.value"
          :error-message="moveEvaluation.errorMessage.value"
        />
      </section>
    </div>
  </main>
</template>

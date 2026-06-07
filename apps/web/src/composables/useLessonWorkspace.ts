import { computed, onMounted, ref } from 'vue'
import type { Lesson, LessonSummary } from '../api/types'
import type { LessonMove } from '../core/xiangqi'
import { nextMoveTargetsForActiveNode } from '../core/movePreview'
import { lessonPrincipleGroups } from '../content/principles'
import { useLessonPlayer } from './useLessonPlayer'
import { useLessons } from './useLessons'
import { useMoveEvaluation } from './useMoveEvaluation'

export type LessonMobileTab = 'board' | 'graph' | 'info'

export interface MobileTab {
  id: LessonMobileTab
  label: string
}

const MOBILE_TABS: ReadonlyArray<MobileTab> = [
  { id: 'board', label: 'Bàn cờ' },
  { id: 'graph', label: 'Biến' },
  { id: 'info', label: 'Lý thuyết' },
]

const COMBINED_PATH = '/combined'

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

function detectCombinedPage(): boolean {
  if (typeof window === 'undefined') return false
  return window.location.pathname.replace(/\/$/, '') === COMBINED_PATH
}

function nextMoveSignature(move: LessonMove): string {
  return `${move.side}:${move.from.file},${move.from.rank}:${move.to.file},${move.to.rank}`
}

export function useLessonWorkspace() {
  const catalog = useLessons()
  const lesson = computed<Lesson>(() => catalog.activeLesson.value ?? emptyLesson)
  const player = useLessonPlayer(lesson)
  const moveEvaluation = useMoveEvaluation(player)

  const isPrinciplesExpanded = ref(true)
  const isSidebarOpen = ref(false)
  const isCombinedPage = ref(detectCombinedPage())
  const activeMobileTab = ref<LessonMobileTab>('board')

  const principleCount = computed(() =>
    lessonPrincipleGroups.reduce((count, group) => count + group.items.length, 0),
  )

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
      lesson.value.lines
        .map((line) => line.moves?.[choicePromptPly.value]?.id)
        .filter((moveId): moveId is string => Boolean(moveId)),
    )

    return lesson.value.choice.options.filter((option) => moveIdsAtPrompt.has(option.moveId))
  })

  const shouldShowChoice = computed(
    () =>
      Boolean(lesson.value.choice.prompt) &&
      player.activeMoveIndex.value === choicePromptPly.value &&
      visibleChoiceOptions.value.length > 1,
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

  function toggleSidebar() {
    isSidebarOpen.value = !isSidebarOpen.value
  }

  function togglePrinciples() {
    isPrinciplesExpanded.value = !isPrinciplesExpanded.value
  }

  function selectLessonAndResetTab(id: string) {
    catalog.selectLesson(id)
    isSidebarOpen.value = false
    activeMobileTab.value = 'board'
  }

  function goToGraphMove(lineId: string, index: number) {
    player.setLine(lineId)
    player.goToMove(index)
  }

  function goToPreviewMove(move: LessonMove) {
    const target = nextMoveTargets.value.find(
      (item) => nextMoveSignature(item.move) === nextMoveSignature(move),
    )
    if (!target) return
    goToGraphMove(target.lineId, player.activeMoveIndex.value + 1)
  }

  function lessonsForCategory(category: string): LessonSummary[] {
    return catalog.summaries.value.filter((item) => item.category === category)
  }

  onMounted(() => {
    if (!isCombinedPage.value) {
      catalog.loadCatalog()
    }
  })

  return {
    // catalog
    activeCategory: catalog.activeCategory,
    activeLesson: catalog.activeLesson,
    activeLessonId: catalog.activeLessonId,
    categoryStats: catalog.categoryStats,
    errorMessage: catalog.errorMessage,
    isCategoryExpanded: catalog.isCategoryExpanded,
    isLoading: catalog.isLoading,
    loadCatalog: catalog.loadCatalog,
    selectLesson: catalog.selectLesson,
    summaries: catalog.summaries,
    toggleCategory: catalog.toggleCategory,
    // workspace state
    lesson,
    player,
    moveEvaluation,
    isPrinciplesExpanded,
    isSidebarOpen,
    isCombinedPage,
    activeMobileTab,
    mobileTabs: MOBILE_TABS,
    principleCount,
    // lesson play helpers
    choiceFeedback,
    activeMoveComment,
    nextMovePreviews,
    canGoPrevious,
    canGoNext,
    shouldShowChoice,
    visibleChoiceOptions,
    // actions
    toggleSidebar,
    togglePrinciples,
    selectLessonAndResetTab,
    goToGraphMove,
    goToPreviewMove,
    lessonsForCategory,
  }
}

import { computed, ref, watch } from 'vue'
import { fetchCategories, fetchLesson, fetchLessons } from '../api/client'
import type { Lesson, LessonSummary } from '../api/types'

function lessonIdFromPath() {
  const [, lessonId] = window.location.pathname.match(/^\/lessons?\/([^/]+)\/?$/) ?? []
  return lessonId ? decodeURIComponent(lessonId) : ''
}

function lessonPath(id: string) {
  return `/lessons/${encodeURIComponent(id)}`
}

function addExpandedCategory(expandedCategories: ReturnType<typeof ref<Set<string>>>, category: string) {
  if (!category) return
  const next = new Set(expandedCategories.value)
  next.add(category)
  expandedCategories.value = next
}

function categoryRank(category: string) {
  const normalized = category.trim().toLowerCase()

  if (normalized.includes('cạm bẫy')) return 1
  if (normalized.includes('khai cuộc')) return 0
  if (normalized.includes('trung cuộc')) return 2
  if (normalized.includes('tàn cuộc')) return 3

  return 4
}

function sortCategories(categories: string[]) {
  return [...categories].sort((left, right) => {
    const rankDelta = categoryRank(left) - categoryRank(right)

    return rankDelta || left.localeCompare(right, 'vi')
  })
}

function sortLessonSummaries(summaries: LessonSummary[]) {
  return [...summaries].sort((left, right) => {
    const rankDelta = categoryRank(left.category) - categoryRank(right.category)

    return rankDelta || left.category.localeCompare(right.category, 'vi')
  })
}

export function useLessons() {
  const categories = ref<string[]>([])
  const summaries = ref<LessonSummary[]>([])
  const activeCategory = ref('')
  const activeLessonId = ref('')
  const activeLesson = ref<Lesson | null>(null)
  const expandedCategories = ref(new Set<string>())
  const routeLessonId = ref(lessonIdFromPath())
  const isLoading = ref(true)
  const errorMessage = ref('')

  const categoryStats = computed(() =>
    categories.value.map((name) => ({
      name,
      count: summaries.value.filter((lesson) => lesson.category === name).length,
    })),
  )

  async function loadCatalog() {
    isLoading.value = true
    errorMessage.value = ''

    try {
      const [nextCategories, nextSummaries] = await Promise.all([fetchCategories(), fetchLessons()])
      const sortedCategories = sortCategories(nextCategories)
      const sortedSummaries = sortLessonSummaries(nextSummaries)

      categories.value = sortedCategories
      summaries.value = sortedSummaries
      const routeLesson = sortedSummaries.find((lesson) => lesson.id === routeLessonId.value)
      const currentLesson = sortedSummaries.find((lesson) => lesson.id === activeLessonId.value)
      const nextLesson = routeLesson ?? currentLesson ?? sortedSummaries[0]

      activeCategory.value = nextLesson?.category ?? sortedCategories[0] ?? ''
      activeLessonId.value = nextLesson?.id ?? ''
      addExpandedCategory(expandedCategories, activeCategory.value)

      if (activeLessonId.value && routeLessonId.value !== activeLessonId.value) {
        window.history.replaceState({}, '', lessonPath(activeLessonId.value))
      }
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : 'Không thể tải dữ liệu bài học.'
    } finally {
      isLoading.value = false
    }
  }

  async function selectLesson(id: string) {
    if (!id) return
    const lesson = summaries.value.find((item) => item.id === id)
    if (lesson) {
      activeCategory.value = lesson.category
      addExpandedCategory(expandedCategories, lesson.category)
    }
    activeLessonId.value = id
    if (window.location.pathname !== lessonPath(id)) {
      window.history.pushState({}, '', lessonPath(id))
    }
  }

  function toggleCategory(category: string) {
    const next = new Set(expandedCategories.value)
    if (next.has(category)) {
      next.delete(category)
    } else {
      next.add(category)
    }
    expandedCategories.value = next
  }

  function isCategoryExpanded(category: string) {
    return expandedCategories.value.has(category)
  }

  function syncLessonFromPath() {
    routeLessonId.value = lessonIdFromPath()
    if (!routeLessonId.value) return

    const lesson = summaries.value.find((item) => item.id === routeLessonId.value)
    if (!lesson) return

    activeCategory.value = lesson.category
    activeLessonId.value = lesson.id
    addExpandedCategory(expandedCategories, lesson.category)
  }

  window.addEventListener('popstate', syncLessonFromPath)

  watch(
    activeLessonId,
    async (lessonId) => {
      if (!lessonId) {
        activeLesson.value = null
        return
      }

      isLoading.value = true
      errorMessage.value = ''

      try {
        activeLesson.value = await fetchLesson(lessonId)
      } catch (error) {
        errorMessage.value = error instanceof Error ? error.message : 'Không thể tải bài học.'
      } finally {
        isLoading.value = false
      }
    },
    { immediate: false },
  )

  return {
    activeCategory,
    activeLesson,
    activeLessonId,
    categories,
    categoryStats,
    errorMessage,
    isLoading,
    loadCatalog,
    isCategoryExpanded,
    selectLesson,
    toggleCategory,
    summaries,
  }
}

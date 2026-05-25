import { computed, ref, watch } from 'vue'
import { fetchCategories, fetchLesson, fetchLessons } from '../api/client'
import type { Lesson, LessonSummary } from '../api/types'

export function useLessons() {
  const categories = ref<string[]>([])
  const summaries = ref<LessonSummary[]>([])
  const activeCategory = ref('')
  const activeLessonId = ref('')
  const activeLesson = ref<Lesson | null>(null)
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
      categories.value = nextCategories
      summaries.value = nextSummaries
      activeCategory.value = activeCategory.value || nextCategories[0] || ''
      activeLessonId.value = activeLessonId.value || nextSummaries.find((lesson) => lesson.category === activeCategory.value)?.id || nextSummaries[0]?.id || ''
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : 'Không thể tải dữ liệu bài học.'
    } finally {
      isLoading.value = false
    }
  }

  async function selectLesson(id: string) {
    if (!id) return
    const lesson = summaries.value.find((item) => item.id === id)
    if (lesson) activeCategory.value = lesson.category
    activeLessonId.value = id
  }

  function selectCategory(category: string) {
    activeCategory.value = category
    activeLessonId.value = summaries.value.find((lesson) => lesson.category === category)?.id || summaries.value[0]?.id || ''
  }

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
    selectCategory,
    selectLesson,
    summaries,
  }
}

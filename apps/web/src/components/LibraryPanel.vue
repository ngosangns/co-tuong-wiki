<script setup lang="ts">
import { BookOpen } from '@lucide/vue'
import { computed } from 'vue'
import Input from './ui/input.vue'
import { Menu, Search, X, GitBranch } from '@lucide/vue'

interface CategoryStat {
  name: string
  count: number
}

interface LessonSummary {
  id: string
  title: string
  category: string
  difficulty: string
}

const props = defineProps<{
  isOpen: boolean
  lessonTitle: string
  categoryStats: CategoryStat[]
  activeCategory: string
  activeLessonId: string
  summaries: LessonSummary[]
  isCategoryExpanded: (category: string) => boolean
}>()

const emit = defineEmits<{
  toggle: []
  close: []
  selectLesson: [id: string]
  toggleCategory: [category: string]
}>()

const summaryByCategory = computed(() => {
  const map = new Map<string, LessonSummary[]>()
  for (const item of props.summaries) {
    const list = map.get(item.category) ?? []
    list.push(item)
    map.set(item.category, list)
  }
  return map
})
</script>

<template>
  <header class="mobile-header">
    <button
      type="button"
      class="mobile-nav-toggle"
      aria-label="Mở danh sách bài học"
      :aria-expanded="props.isOpen"
      @click="emit('toggle')"
    >
      <Menu v-if="!props.isOpen" :size="20" aria-hidden="true" />
      <X v-else :size="20" aria-hidden="true" />
      <span>Danh mục</span>
    </button>
    <h2 class="mobile-lesson-title">{{ props.lessonTitle }}</h2>
  </header>

  <aside class="library-panel" :class="{ 'is-open': props.isOpen }">
    <div class="brand-lockup">
      <div class="brand-mark">象</div>
      <div>
        <p class="eyebrow">Cờ Tướng Wiki</p>
        <h1>Học bằng thế cờ thật</h1>
      </div>
    </div>

    <div
      class="flex items-center gap-2 rounded-md border border-border bg-panel-strong px-3 py-2.5 text-muted-foreground transition-colors focus-within:border-primary focus-within:ring-2 focus-within:ring-ring"
    >
      <Search :size="18" aria-hidden="true" />
      <Input
        type="search"
        placeholder="Tìm khai cuộc, cạm bẫy..."
        class="border-0 bg-transparent p-0 text-foreground placeholder:text-muted-foreground focus-visible:ring-0 focus-visible:ring-offset-0"
      />
    </div>

    <a class="combined-nav-link" href="/combined">
      <GitBranch :size="18" aria-hidden="true" />
      <span>Tổng hợp toàn bộ lesson</span>
    </a>

    <section class="topic-list" aria-label="Chủ đề và bài học">
      <div v-for="category in props.categoryStats" :key="category.name" class="topic-group">
        <button
          class="topic"
          :class="{
            active: props.activeCategory === category.name,
            expanded: props.isCategoryExpanded(category.name),
          }"
          type="button"
          :aria-expanded="props.isCategoryExpanded(category.name)"
          @click="emit('toggleCategory', category.name)"
        >
          <BookOpen :size="18" aria-hidden="true" />
          <span>{{ category.name }}</span>
          <small>{{ category.count }}</small>
        </button>

        <div v-if="props.isCategoryExpanded(category.name)" class="lesson-child-list">
          <button
            v-for="(item, lessonIndex) in summaryByCategory.get(category.name) ?? []"
            :key="item.id"
            type="button"
            class="lesson-child"
            :class="{ active: props.activeLessonId === item.id }"
            @click="emit('selectLesson', item.id)"
          >
            <small class="lesson-child-index">{{ lessonIndex + 1 }}</small>
            <span>{{ item.title }}</span>
          </button>
        </div>
      </div>
    </section>
  </aside>
</template>

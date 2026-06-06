<script setup lang="ts">
import { GitBranch, Gauge } from '@lucide/vue'
import type { LessonMobileTab } from '../composables/useLessonWorkspace'

defineProps<{
  activeTab: LessonMobileTab
}>()

const emit = defineEmits<{
  'update:activeTab': [tab: LessonMobileTab]
}>()
</script>

<template>
  <nav class="lesson-mobile-dock" aria-label="Điều hướng nhanh">
    <button
      type="button"
      class="lesson-mobile-dock-button"
      :class="{ active: activeTab === 'graph' }"
      :aria-pressed="activeTab === 'graph'"
      @click="emit('update:activeTab', 'graph')"
    >
      <GitBranch :size="16" aria-hidden="true" />
      <span>Biến</span>
    </button>
    <button
      type="button"
      class="lesson-mobile-dock-button"
      :class="{ active: activeTab === 'info' }"
      :aria-pressed="activeTab === 'info'"
      @click="emit('update:activeTab', 'info')"
    >
      <Gauge :size="16" aria-hidden="true" />
      <span>Lý thuyết</span>
    </button>
  </nav>
</template>

<script setup lang="ts">
import { ChevronDown, Lightbulb } from '@lucide/vue'

defineProps<{
  isExpanded: boolean
  principleCount: number
}>()

const emit = defineEmits<{
  toggle: []
}>()
</script>

<template>
  <section class="principles" :class="{ expanded: isExpanded }">
    <button
      type="button"
      class="principles-toggle"
      :aria-expanded="isExpanded"
      aria-controls="lesson-principles"
      @click="emit('toggle')"
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

    <div v-show="isExpanded" id="lesson-principles" class="principles-groups">
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
</template>

<script lang="ts">
import { lessonPrincipleGroups } from '../content/principles'
export default { name: 'LessonInspectorPanel' }
</script>

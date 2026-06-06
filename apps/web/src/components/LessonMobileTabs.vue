<script setup lang="ts">
import Tabs from './ui/tabs.vue'
import TabsList from './ui/tabs-list.vue'
import TabsTrigger from './ui/tabs-trigger.vue'
import TabsContent from './ui/tabs-content.vue'
import MoveGraph from './MoveGraph.vue'
import LessonInspectorPanel from './LessonInspectorPanel.vue'
import type { Lesson } from '../api/types'
import type { LessonMobileTab, MobileTab } from '../composables/useLessonWorkspace'

const props = defineProps<{
  modelValue: LessonMobileTab
  lesson: Lesson
  activeLineId: string
  activeMoveIndex: number
  isPrinciplesExpanded: boolean
  principleCount: number
  tabs: ReadonlyArray<MobileTab>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: LessonMobileTab]
  selectLine: [lineId: string]
  selectMove: [lineId: string, index: number]
  togglePrinciples: []
}>()

function onTabChange(value: string) {
  emit('update:modelValue', value as LessonMobileTab)
}

function onSelectMove(lineId: string, index: number) {
  emit('selectMove', lineId, index)
}
</script>

<template>
  <Tabs :model-value="props.modelValue" class="mobile-only" @update:model-value="onTabChange">
    <TabsList class="grid w-full grid-cols-3">
      <TabsTrigger v-for="tab in props.tabs" :key="tab.id" :value="tab.id">
        {{ tab.label }}
      </TabsTrigger>
    </TabsList>

    <TabsContent value="graph" class="board-side-panel mobile-tab-panel" aria-label="Điều khiển và phản hồi bài học">
      <MoveGraph
        :lines="props.lesson.lines"
        :active-line-id="props.activeLineId"
        :active-move-index="props.activeMoveIndex"
        :initial-fen="props.lesson.initialFen"
        @select-line="emit('selectLine', $event)"
        @select-move="onSelectMove"
      />
    </TabsContent>

    <TabsContent value="info" class="lesson-panel mobile-tab-panel" aria-label="Nội dung bài học">
      <LessonInspectorPanel
        :is-expanded="props.isPrinciplesExpanded"
        :principle-count="props.principleCount"
        @toggle="emit('togglePrinciples')"
      />
    </TabsContent>
  </Tabs>
</template>

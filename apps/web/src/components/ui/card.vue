<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

type CardVariant = 'elevated' | 'flat' | 'outlined'

const props = withDefaults(
  defineProps<{
    variant?: CardVariant
    interactive?: boolean
    as?: keyof HTMLElementTagNameMap
  }>(),
  { variant: 'flat', interactive: false, as: 'section' },
)

const classes = computed(() => cn('card', `card-${props.variant}`, { 'card-interactive': props.interactive }))
</script>

<template>
  <component :is="props.as" :class="classes">
    <slot />
  </component>
</template>

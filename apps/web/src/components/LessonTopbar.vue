<script setup lang="ts">
defineProps<{
  title: string
  category: string
  difficulty: string
}>()
</script>

<template>
  <section class="lesson-topbar" aria-label="Bài học">
    <article class="wiki-article">
      <p class="eyebrow">{{ category }} · {{ difficulty }}</p>
      <h2>{{ title }}</h2>
    </article>
  </section>
</template>

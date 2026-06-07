<script setup lang="ts">
import { BookOpen, Sparkles } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import { fetchOpeningBook } from '../api/client'
import type { OpeningBookResponse } from '../api/types'
import type { BoardState, Side } from '../core/xiangqi'
import { boardToXiangqiFen } from '../engine/fen'

const props = defineProps<{
  board: BoardState
  sideToMove: Side
  ply: number
  openingDepth: number
}>()

const emit = defineEmits<{
  selectMove: [
    notation: { from: { file: number; rank: number }; to: { file: number; rank: number }; notation: string },
  ]
}>()

const book = ref<OpeningBookResponse | null>(null)
const isLoading = ref(false)
const errorMessage = ref('')

const fen = computed(() => boardToXiangqiFen(props.board, props.sideToMove, Math.floor(props.ply / 2) + 1))

async function loadBook() {
  isLoading.value = true
  errorMessage.value = ''
  try {
    book.value = await fetchOpeningBook(fen.value)
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Không thể tra cứu opening book.'
    book.value = null
  } finally {
    isLoading.value = false
  }
}

watch(
  fen,
  () => {
    void loadBook()
  },
  { immediate: true },
)

const visible = computed(() => {
  if (!book.value) return []
  return book.value.moves.filter((m) => m.popularity >= 0.005).slice(0, 6)
})

const isAvailable = computed(() => (book.value?.available ?? false) && (book.value?.moves.length ?? 0) > 0)

function parseMoveKey(key: string) {
  const [fromPart, toPart] = key.split('->')
  if (!fromPart || !toPart) return null
  const [fromFile, fromRank] = fromPart.split(',').map(Number)
  const [toFile, toRank] = toPart.split(',').map(Number)
  if ([fromFile, fromRank, toFile, toRank].some((n) => Number.isNaN(n))) return null
  return {
    from: { file: fromFile, rank: fromRank },
    to: { file: toFile, rank: toRank },
  }
}

function playMove(move: { move: string; notation: string }) {
  const parsed = parseMoveKey(move.move)
  if (!parsed) return
  emit('selectMove', { ...parsed, notation: move.notation })
}

function totalGames() {
  if (!book.value) return 0
  return book.value.moves.reduce((sum, m) => sum + m.frequency, 0)
}

const showHeader = computed(() => props.openingDepth > 0 && props.ply <= props.openingDepth)
</script>

<template>
  <section v-if="showHeader" class="opening-suggestions" aria-label="Gợi ý khai cuộc">
    <header class="opening-suggestions-header">
      <div class="opening-suggestions-title">
        <BookOpen :size="16" aria-hidden="true" />
        <div>
          <h4>Opening book</h4>
          <p v-if="book?.visits">
            Đã thấy trong <strong>{{ book.visits }}</strong> ván
          </p>
          <p v-else>Đang tra cứu…</p>
        </div>
      </div>
      <span v-if="isAvailable" class="opening-suggestions-stats">
        <Sparkles :size="14" aria-hidden="true" />
        {{ totalGames() }} nước đi cùng vị trí
      </span>
    </header>

    <div v-if="isLoading" class="opening-suggestions-placeholder">Đang tra cứu…</div>
    <div v-else-if="errorMessage" class="opening-suggestions-placeholder error">{{ errorMessage }}</div>
    <div v-else-if="!isAvailable" class="opening-suggestions-placeholder">
      Chưa có dữ liệu opening book cho vị trí này.
    </div>
    <ol v-else class="opening-suggestions-list">
      <li
        v-for="move in visible"
        :key="move.move"
        class="opening-suggestion"
        :class="{ 'is-popular': move.popularity >= 0.5 }"
      >
        <button type="button" class="opening-suggestion-button" @click="playMove(move)">
          <span class="opening-suggestion-notation">{{ move.notation }}</span>
          <span v-if="move.name" class="opening-suggestion-name">{{ move.name }}</span>
          <span class="opening-suggestion-bar" aria-hidden="true">
            <span
              class="opening-suggestion-bar-fill"
              :style="{ width: `${Math.max(8, move.popularity * 100)}%` }"
            ></span>
          </span>
          <span class="opening-suggestion-popularity">{{ Math.round(move.popularity * 100) }}%</span>
        </button>
      </li>
    </ol>
  </section>
</template>

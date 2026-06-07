<script setup lang="ts">
import { computed, ref } from 'vue'
import type { BoardState, Coordinate, LessonMove, Piece } from '../core/xiangqi'
import { sameSquare } from '../core/xiangqi'

const props = defineProps<{
  board: BoardState
  currentMove?: LessonMove
  previewMoves?: LessonMove[]
  engineSuggestion?: LessonMove
}>()

const emit = defineEmits<{
  selectPreviewMove: [move: LessonMove]
  selectEngineMove: [move: LessonMove]
  swipeLeft: []
  swipeRight: []
}>()

const touchStartX = ref(0)
const touchStartY = ref(0)
const SWIPE_THRESHOLD = 50
const VERTICAL_TOLERANCE = 80

function onTouchStart(event: TouchEvent) {
  touchStartX.value = event.changedTouches[0].screenX
  touchStartY.value = event.changedTouches[0].screenY
}

function onTouchEnd(event: TouchEvent) {
  const endX = event.changedTouches[0].screenX
  const endY = event.changedTouches[0].screenY
  const deltaX = endX - touchStartX.value
  const deltaY = endY - touchStartY.value

  if (Math.abs(deltaY) > VERTICAL_TOLERANCE) return

  if (deltaX < -SWIPE_THRESHOLD) {
    emit('swipeLeft')
  } else if (deltaX > SWIPE_THRESHOLD) {
    emit('swipeRight')
  }
}

const ranks = Array.from({ length: 10 }, (_, rank) => rank)
const files = Array.from({ length: 9 }, (_, file) => file)
const redFileLabels = files.map((file) => String(file + 1))
const blackFileLabels = files.map((file) => String(9 - file))

const currentFrom = computed(() => props.currentMove?.from)
const currentTo = computed(() => props.currentMove?.to)
const movePreviews = computed(() => {
  const seen = new Set<string>()

  return (props.previewMoves ?? []).flatMap((move, index) => {
    const piece = props.board.find((item) => item.side === move.side && sameSquare(item.position, move.from))
    if (!piece) return []

    const key = `${move.side}:${move.from.file},${move.from.rank}:${move.to.file},${move.to.rank}`
    if (seen.has(key)) return []

    seen.add(key)
    return [{ id: `${key}:${index}`, move, piece, arrow: previewArrow(move) }]
  })
})
const previewFromSquares = computed(
  () => new Set(movePreviews.value.map((preview) => squareKey(preview.move.from))),
)
const previewToSquares = computed(
  () => new Set(movePreviews.value.map((preview) => squareKey(preview.move.to))),
)

const engineSuggestionPreview = computed(() => {
  const suggestion = props.engineSuggestion
  if (!suggestion) return null
  const piece = props.board.find(
    (item) => item.side === suggestion.side && sameSquare(item.position, suggestion.from),
  )
  if (!piece) return null
  return {
    id: 'engine-suggestion',
    move: suggestion,
    piece,
    arrow: previewArrow(suggestion),
  }
})

const engineMatchesPlayed = computed(() => {
  const suggestion = props.engineSuggestion
  if (!suggestion) return false
  return movePreviews.value.some(
    (preview) => sameSquare(preview.move.from, suggestion.from) && sameSquare(preview.move.to, suggestion.to),
  )
})

function squareKey(coordinate: Coordinate) {
  return `${coordinate.file}:${coordinate.rank}`
}

function squareState(coordinate: Coordinate) {
  const isFrom = currentFrom.value ? sameSquare(coordinate, currentFrom.value) : false
  const isTo = currentTo.value ? sameSquare(coordinate, currentTo.value) : false

  return {
    isFrom,
    isTo,
    isPreviewFrom: previewFromSquares.value.has(squareKey(coordinate)),
    isPreviewTo: previewToSquares.value.has(squareKey(coordinate)),
    isPalace: coordinate.file >= 3 && coordinate.file <= 5 && (coordinate.rank <= 2 || coordinate.rank >= 7),
    isRiverBorder: coordinate.rank === 4,
  }
}

function pieceStyle(piece: Piece) {
  return {
    transform: `translate(${piece.position.file * 100}%, ${piece.position.rank * 100}%)`,
  }
}

function coordinateStyle(coordinate: Coordinate) {
  return {
    transform: `translate(${coordinate.file * 100}%, ${coordinate.rank * 100}%)`,
  }
}

function boardPoint(coordinate: Coordinate) {
  return {
    x: coordinate.file * 100 + 50,
    y: coordinate.rank * 100 + 50,
  }
}

function previewArrow(move: LessonMove) {
  const from = boardPoint(move.from)
  const to = boardPoint(move.to)
  const dx = to.x - from.x
  const dy = to.y - from.y
  const length = Math.hypot(dx, dy)

  if (!length) return { x1: from.x, y1: from.y, x2: to.x, y2: to.y }

  const unitX = dx / length
  const unitY = dy / length
  const sourceOffset = 28
  const targetOffset = 38

  return {
    x1: from.x + unitX * sourceOffset,
    y1: from.y + unitY * sourceOffset,
    x2: to.x - unitX * targetOffset,
    y2: to.y - unitY * targetOffset,
  }
}
</script>

<template>
  <div
    class="xiangqi-board-shell"
    aria-label="Bàn cờ tướng"
    v-motion
    :initial="{ opacity: 0, scale: 0.96 }"
    :enter="{ opacity: 1, scale: 1, transition: { duration: 600, ease: 'easeOut' } }"
    @touchstart="onTouchStart"
    @touchend="onTouchEnd"
  >
    <div class="board-file-labels board-file-labels-top" aria-label="Cột bên Đen">
      <span v-for="label in blackFileLabels" :key="`black-file-${label}`">{{ label }}</span>
    </div>
    <div class="xiangqi-board">
      <div class="board-grid">
        <template v-for="rank in ranks" :key="rank">
          <button
            v-for="file in files"
            :key="`${file}-${rank}`"
            class="board-square"
            :class="{
              'is-palace': squareState({ file, rank }).isPalace,
              'is-from': squareState({ file, rank }).isFrom,
              'is-to': squareState({ file, rank }).isTo,
              'is-preview-from': squareState({ file, rank }).isPreviewFrom,
              'is-preview-to': squareState({ file, rank }).isPreviewTo,
              'is-river-border': squareState({ file, rank }).isRiverBorder,
            }"
            type="button"
            :aria-label="`Ô Đỏ ${file + 1}-${9 - rank}, Đen ${9 - file}-${rank}`"
          ></button>
        </template>
      </div>
      <div class="board-piece-layer" aria-hidden="true">
        <span v-for="piece in board" :key="piece.id" class="piece-slot" :style="pieceStyle(piece)">
          <span class="piece" :class="piece.side">
            {{ piece.label }}
          </span>
        </span>
      </div>
      <div v-if="movePreviews.length" class="board-preview-layer" aria-label="Nước tiếp theo">
        <svg class="preview-arrow-layer" viewBox="0 0 900 1000" preserveAspectRatio="none" aria-hidden="true">
          <defs>
            <marker
              id="preview-arrow-head"
              markerWidth="32"
              markerHeight="32"
              refX="28"
              refY="16"
              orient="auto"
              markerUnits="userSpaceOnUse"
            >
              <path class="preview-arrow-head" d="M 0 0 L 32 16 L 0 32 z"></path>
            </marker>
            <marker
              id="engine-arrow-head"
              markerWidth="32"
              markerHeight="32"
              refX="28"
              refY="16"
              orient="auto"
              markerUnits="userSpaceOnUse"
            >
              <path class="engine-arrow-head" d="M 0 0 L 32 16 L 0 32 z"></path>
            </marker>
          </defs>
          <line
            v-for="preview in movePreviews"
            :key="`${preview.id}:arrow`"
            class="preview-arrow"
            :x1="preview.arrow.x1"
            :y1="preview.arrow.y1"
            :x2="preview.arrow.x2"
            :y2="preview.arrow.y2"
            marker-end="url(#preview-arrow-head)"
          ></line>
          <line
            v-if="engineSuggestionPreview && !engineMatchesPlayed"
            :key="`engine-suggestion:arrow`"
            class="engine-suggestion-arrow"
            :x1="engineSuggestionPreview.arrow.x1"
            :y1="engineSuggestionPreview.arrow.y1"
            :x2="engineSuggestionPreview.arrow.x2"
            :y2="engineSuggestionPreview.arrow.y2"
            marker-end="url(#engine-arrow-head)"
          ></line>
        </svg>
        <button
          v-for="preview in movePreviews"
          :key="`${preview.id}:source`"
          class="preview-source-slot"
          type="button"
          :style="coordinateStyle(preview.move.from)"
          :aria-label="`Chọn preview từ ${preview.move.from.file}-${preview.move.from.rank}`"
          @click="emit('selectPreviewMove', preview.move)"
        >
          <span class="preview-source-ring" aria-hidden="true"></span>
        </button>
        <button
          v-for="preview in movePreviews"
          :key="`${preview.id}:target`"
          class="preview-piece-slot"
          type="button"
          :style="coordinateStyle(preview.move.to)"
          :aria-label="`Chọn preview đến ${preview.move.to.file}-${preview.move.to.rank}`"
          @click="emit('selectPreviewMove', preview.move)"
        >
          <span class="preview-piece" :class="preview.piece.side" aria-hidden="true">
            {{ preview.piece.label }}
          </span>
        </button>
        <template v-if="engineSuggestionPreview && !engineMatchesPlayed">
          <button
            class="engine-suggestion-source"
            type="button"
            :style="coordinateStyle(engineSuggestionPreview.move.from)"
            :aria-label="`Engine đề xuất từ ${engineSuggestionPreview.move.from.file}-${engineSuggestionPreview.move.from.rank}`"
            @click="emit('selectEngineMove', engineSuggestionPreview.move)"
          >
            <span class="engine-suggestion-ring" aria-hidden="true"></span>
          </button>
          <button
            class="engine-suggestion-target"
            type="button"
            :style="coordinateStyle(engineSuggestionPreview.move.to)"
            :aria-label="`Engine đề xuất đến ${engineSuggestionPreview.move.to.file}-${engineSuggestionPreview.move.to.rank}`"
            @click="emit('selectEngineMove', engineSuggestionPreview.move)"
          >
            <span
              class="engine-suggestion-piece"
              :class="engineSuggestionPreview.piece.side"
              aria-hidden="true"
            >
              {{ engineSuggestionPreview.piece.label }}
            </span>
          </button>
        </template>
      </div>
    </div>
    <div class="board-file-labels board-file-labels-bottom" aria-label="Cột bên Đỏ">
      <span v-for="label in redFileLabels" :key="`red-file-${label}`">{{ label }}</span>
    </div>
  </div>
</template>

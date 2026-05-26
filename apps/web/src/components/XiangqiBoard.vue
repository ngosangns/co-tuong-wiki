<script setup lang="ts">
import { computed } from 'vue'
import type { BoardState, Coordinate, LessonMove, Piece } from '../core/xiangqi'
import { sameSquare } from '../core/xiangqi'

const props = defineProps<{
  board: BoardState
  currentMove?: LessonMove
}>()

const ranks = Array.from({ length: 10 }, (_, rank) => rank)
const files = Array.from({ length: 9 }, (_, file) => file)
const redFileLabels = files.map((file) => String(file + 1))
const blackFileLabels = files.map((file) => String(9 - file))

const currentFrom = computed(() => props.currentMove?.from)
const currentTo = computed(() => props.currentMove?.to)

function squareState(coordinate: Coordinate) {
  const isFrom = currentFrom.value ? sameSquare(coordinate, currentFrom.value) : false
  const isTo = currentTo.value ? sameSquare(coordinate, currentTo.value) : false

  return {
    isFrom,
    isTo,
    isPalace:
      coordinate.file >= 3 &&
      coordinate.file <= 5 &&
      (coordinate.rank <= 2 || coordinate.rank >= 7),
    isRiverBorder: coordinate.rank === 4,
  }
}

function pieceStyle(piece: Piece) {
  return {
    transform: `translate(${piece.position.file * 100}%, ${piece.position.rank * 100}%)`,
  }
}
</script>

<template>
  <div class="xiangqi-board-shell" aria-label="Bàn cờ tướng">
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
    </div>
    <div class="board-file-labels board-file-labels-bottom" aria-label="Cột bên Đỏ">
      <span v-for="label in redFileLabels" :key="`red-file-${label}`">{{ label }}</span>
    </div>
  </div>
</template>

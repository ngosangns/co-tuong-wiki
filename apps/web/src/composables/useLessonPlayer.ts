import { computed, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import type { Lesson, LessonChoice } from '../api/types'
import { initialBoard, replayMoves } from '../core/xiangqi'
import { boardFromXiangqiFen } from '../engine/fen'

export function useLessonPlayer(lessonSource: MaybeRefOrGetter<Lesson>) {
  const lesson = computed(() => toValue(lessonSource))
  const activeLineId = ref(lesson.value.lines[0]?.id ?? '')
  const activeMoveIndex = ref(0)
  const selectedChoice = ref<LessonChoice['options'][number] | null>(null)

  const activeLine = computed(() => lesson.value.lines.find((line) => line.id === activeLineId.value) ?? lesson.value.lines[0])
  const activeMoves = computed(() => activeLine.value?.moves ?? [])
  const visibleMoves = computed(() => activeMoves.value.slice(0, activeMoveIndex.value))
  const activeInitialFen = computed(() => activeLine.value?.initialFen ?? lesson.value.initialFen)
  const initialLessonBoard = computed(() => (activeInitialFen.value ? boardFromXiangqiFen(activeInitialFen.value) : initialBoard))
  const board = computed(() => replayMoves(initialLessonBoard.value, visibleMoves.value))
  const currentMove = computed(() => activeMoves.value[activeMoveIndex.value - 1])

  watch(
    lesson,
    (nextLesson) => {
      activeLineId.value = nextLesson.lines[0]?.id ?? ''
      activeMoveIndex.value = 0
      selectedChoice.value = null
    },
  )

  function setLine(lineId: string) {
    activeLineId.value = lineId
    activeMoveIndex.value = Math.min(activeMoveIndex.value, activeMoves.value.length)
    selectedChoice.value = null
  }

  function goToMove(index: number) {
    activeMoveIndex.value = Math.max(0, Math.min(index, activeMoves.value.length))
    selectedChoice.value = null
  }

  function previous() {
    goToMove(activeMoveIndex.value - 1)
  }

  function next() {
    goToMove(activeMoveIndex.value + 1)
  }

  function chooseMove(moveId: string) {
    const option = lesson.value.choice.options.find((item) => item.moveId === moveId) ?? null
    const matchingLine = lesson.value.lines.find((line) => line.moves?.some((move) => move.id === moveId))

    if (matchingLine) {
      activeLineId.value = matchingLine.id
      activeMoveIndex.value = (matchingLine.moves?.findIndex((move) => move.id === moveId) ?? -1) + 1
    }

    selectedChoice.value = option
  }

  return {
    activeLine,
    activeLineId,
    activeMoveIndex,
    activeMoves,
    board,
    currentMove,
    selectedChoice,
    setLine,
    goToMove,
    previous,
    next,
    chooseMove,
  }
}

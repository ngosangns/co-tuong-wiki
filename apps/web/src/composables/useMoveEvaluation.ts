import { computed, ref, watch } from 'vue'
import { analyzePosition } from '../api/client'
import { boardToXiangqiFen } from '../engine/fen'
import type { BoardState, LessonMove, Side } from '../core/xiangqi'
import type { EngineEvaluation, EngineStatus } from '../engine/types'

interface MoveEvaluationSource {
  board: { readonly value: BoardState }
  activeMoveIndex: { readonly value: number }
  activeMoves: { readonly value: LessonMove[] }
  currentMove: { readonly value: LessonMove | undefined }
}

function inferSideToMove(moves: LessonMove[], ply: number): Side | null {
  const nextMove = moves[ply]
  if (nextMove) return nextMove.side

  const lastMove = moves[ply - 1]
  if (!lastMove) return moves[0]?.side ?? 'red'
  return lastMove.side === 'red' ? 'black' : 'red'
}

export function useMoveEvaluation(source: MoveEvaluationSource) {
  const status = ref<EngineStatus>('idle')
  const evaluation = ref<EngineEvaluation | null>(null)
  const errorMessage = ref('')

  const nextMove = computed(() => source.activeMoves.value[source.activeMoveIndex.value])
  const sideToMove = computed(() => inferSideToMove(source.activeMoves.value, source.activeMoveIndex.value))
  const fen = computed(() => {
    if (!sideToMove.value) return ''
    return boardToXiangqiFen(source.board.value, sideToMove.value, Math.floor(source.activeMoveIndex.value / 2) + 1)
  })

  watch(
    [fen, nextMove],
    ([nextFen], _previous, onCleanup) => {
      const analysisSide = sideToMove.value

      if (!nextFen || !analysisSide) {
        status.value = 'idle'
        evaluation.value = null
        return
      }

      const controller = new AbortController()
      const timer = window.setTimeout(async () => {
        status.value = 'analyzing'
        errorMessage.value = ''

        try {
          const result = await analyzePosition(
            {
              fen: nextFen,
              sideToMove: analysisSide,
              nextMove: nextMove.value,
            },
            controller.signal,
          )

          if (controller.signal.aborted) return

          // The API response may omit status while still returning a usable evaluation payload.
          evaluation.value = result
          status.value = result.status ?? 'ready'
        } catch (error) {
          if (controller.signal.aborted) return

          status.value = 'error'
          errorMessage.value = error instanceof Error ? error.message : 'Không thể đánh giá vị trí hiện tại.'
        }
      }, 180)

      onCleanup(() => {
        controller.abort()
        window.clearTimeout(timer)
      })
    },
    { immediate: true },
  )

  return {
    status,
    evaluation,
    errorMessage,
    fen,
    nextMove,
    sideToMove,
  }
}

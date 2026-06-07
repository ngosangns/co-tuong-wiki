import { describe, expect, it } from 'vitest'
import type { Lesson, LessonLine } from '../api/types'
import { useLessonPlayer } from './useLessonPlayer'
import { ref } from 'vue'

const makeMove = (overrides: {
  id: string
  side?: 'red' | 'black'
  from: { file: number; rank: number }
  to: { file: number; rank: number }
}) => ({
  id: overrides.id,
  side: overrides.side ?? 'red',
  from: overrides.from,
  to: overrides.to,
  comment: '',
})

const makeLine = (id: string, moves: ReturnType<typeof makeMove>[]): LessonLine => ({
  id,
  title: `Line ${id}`,
  moves,
})

const makeLesson = (lines: LessonLine[], choice: Lesson['choice'] = { prompt: '', options: [] }): Lesson => ({
  id: 'lesson-1',
  title: 'Test lesson',
  category: 'Test',
  difficulty: 'Nhập môn',
  lines,
  choice,
})

describe('useLessonPlayer', () => {
  it('replays moves to the active ply', () => {
    const lesson = ref(
      makeLesson([
        makeLine('l1', [
          makeMove({ id: 'm1', from: { file: 1, rank: 7 }, to: { file: 1, rank: 4 } }),
          makeMove({ id: 'm2', side: 'black', from: { file: 7, rank: 2 }, to: { file: 7, rank: 5 } }),
        ]),
      ]),
    )
    const player = useLessonPlayer(lesson)

    expect(player.board.value).toHaveLength(32)
    expect(player.activeMoveIndex.value).toBe(0)
    expect(player.activeMoves.value).toHaveLength(2)

    player.next()
    expect(player.activeMoveIndex.value).toBe(1)
    expect(player.currentMove.value?.id).toBe('m1')

    player.next()
    expect(player.activeMoveIndex.value).toBe(2)
    expect(player.currentMove.value?.id).toBe('m2')

    player.previous()
    expect(player.activeMoveIndex.value).toBe(1)
  })

  it('previous does not go below ply 0', () => {
    const lesson = ref(
      makeLesson([
        makeLine('l1', [makeMove({ id: 'm1', from: { file: 4, rank: 9 }, to: { file: 4, rank: 8 } })]),
      ]),
    )
    const player = useLessonPlayer(lesson)
    player.previous()
    expect(player.activeMoveIndex.value).toBe(0)
  })

  it('next does not exceed total moves', () => {
    const lesson = ref(
      makeLesson([
        makeLine('l1', [makeMove({ id: 'm1', from: { file: 4, rank: 9 }, to: { file: 4, rank: 8 } })]),
      ]),
    )
    const player = useLessonPlayer(lesson)
    player.next()
    player.next()
    player.next()
    expect(player.activeMoveIndex.value).toBe(1)
  })

  it('setLine switches line and clamps the move index', () => {
    const lineA = makeLine('a', [
      makeMove({ id: 'a1', from: { file: 4, rank: 9 }, to: { file: 4, rank: 8 } }),
    ])
    const lineB = makeLine('b', [
      makeMove({ id: 'b1', from: { file: 1, rank: 7 }, to: { file: 1, rank: 4 } }),
      makeMove({ id: 'b2', side: 'black', from: { file: 7, rank: 2 }, to: { file: 7, rank: 5 } }),
    ])
    const lesson = ref(makeLesson([lineA, lineB]))
    const player = useLessonPlayer(lesson)

    player.next()
    expect(player.activeMoveIndex.value).toBe(1)
    player.setLine('b')
    expect(player.activeLineId.value).toBe('b')
    expect(player.activeMoveIndex.value).toBe(1)
  })

  it('chooseMove jumps to the matching move in any line', () => {
    const lesson = ref(
      makeLesson([
        makeLine('a', [makeMove({ id: 'a1', from: { file: 4, rank: 9 }, to: { file: 4, rank: 8 } })]),
        makeLine('b', [
          makeMove({ id: 'b1', from: { file: 1, rank: 7 }, to: { file: 1, rank: 4 } }),
          makeMove({ id: 'b2', side: 'black', from: { file: 7, rank: 2 }, to: { file: 7, rank: 5 } }),
        ]),
      ]),
    )
    const player = useLessonPlayer(lesson)
    player.chooseMove('b2')
    expect(player.activeLineId.value).toBe('b')
    expect(player.activeMoveIndex.value).toBe(2)
    expect(player.currentMove.value?.id).toBe('b2')
  })

  it('chooseMove sets selectedChoice when option exists in lesson.choice', () => {
    const lesson = ref(
      makeLesson(
        [makeLine('a', [makeMove({ id: 'a1', from: { file: 4, rank: 9 }, to: { file: 4, rank: 8 } })])],
        {
          prompt: 'Chọn nước đi',
          options: [{ moveId: 'a1', label: 'Xe 5 tiến 1', verdict: 'correct', feedback: 'Tốt' }],
        },
      ),
    )
    const player = useLessonPlayer(lesson)
    player.chooseMove('a1')
    expect(player.selectedChoice.value).toEqual({
      moveId: 'a1',
      label: 'Xe 5 tiến 1',
      verdict: 'correct',
      feedback: 'Tốt',
    })
  })

  it('lesson change resets the player to the first line at ply 0', async () => {
    const lesson = ref(
      makeLesson([
        makeLine('a', [makeMove({ id: 'a1', from: { file: 4, rank: 9 }, to: { file: 4, rank: 8 } })]),
      ]),
    )
    const player = useLessonPlayer(lesson)
    player.next()
    expect(player.activeMoveIndex.value).toBe(1)

    lesson.value = makeLesson([
      makeLine('b', [makeMove({ id: 'b1', from: { file: 0, rank: 9 }, to: { file: 0, rank: 8 } })]),
    ])
    await Promise.resolve()
    expect(player.activeLineId.value).toBe('b')
    expect(player.activeMoveIndex.value).toBe(0)
    expect(player.selectedChoice.value).toBeNull()
  })
})

import { describe, expect, it } from 'vitest'
import type { LessonLine } from '../api/types'
import {
  buildActiveNodeId,
  buildGraphPaths,
  buildMoveTrie,
  formatParsedMoveNotation,
  lineMoves,
  moveSignature,
  serializeTrie,
  startKeyForFen,
  startNodeId,
} from './useMoveTrie'

const makeMove = (overrides: {
  id: string
  from: { file: number; rank: number }
  to: { file: number; rank: number }
  side?: 'red' | 'black'
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

describe('useMoveTrie', () => {
  it('moveSignature is stable for same coordinates', () => {
    expect(moveSignature(makeMove({ id: 'a', from: { file: 1, rank: 7 }, to: { file: 1, rank: 4 } }))).toBe(
      moveSignature(makeMove({ id: 'b', from: { file: 1, rank: 7 }, to: { file: 1, rank: 4 } })),
    )
  })

  it('moveSignature differs for different coordinates', () => {
    const a = moveSignature(makeMove({ id: 'a', from: { file: 1, rank: 7 }, to: { file: 1, rank: 4 } }))
    const b = moveSignature(makeMove({ id: 'a', from: { file: 1, rank: 7 }, to: { file: 1, rank: 5 } }))
    expect(a).not.toBe(b)
  })

  it('lineMoves returns an empty array when undefined', () => {
    const line: LessonLine = { id: 'x', title: 'x' }
    expect(lineMoves(line)).toEqual([])
  })

  it('startKeyForFen treats empty FEN and standard FEN as "standard"', () => {
    expect(startKeyForFen('')).toBe('standard')
    expect(startKeyForFen('rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1')).toBe(
      'standard',
    )
    expect(startKeyForFen('4k4/9/9/9/9/9/9/9/9/4K4 b - - 0 1')).not.toBe('standard')
  })

  it('startNodeId encodes start keys safely', () => {
    expect(startNodeId('standard')).toBe('start:standard')
    expect(startNodeId('a/b c')).toBe('start:a%2Fb%20c')
  })

  it('buildGraphPaths replays each line with board state', () => {
    const lineA = makeLine('a', [
      makeMove({ id: 'a1', from: { file: 1, rank: 7 }, to: { file: 1, rank: 4 } }),
      makeMove({ id: 'a2', side: 'black', from: { file: 7, rank: 2 }, to: { file: 7, rank: 5 } }),
    ])
    const paths = buildGraphPaths([lineA])
    expect(paths).toHaveLength(1)
    expect(paths[0]).toHaveLength(2)
    expect(paths[0][1].boardBefore).not.toBeNull()
  })

  it('buildMoveTrie merges lines that share a prefix under one start group', () => {
    const shared = makeMove({ id: 'shared', from: { file: 1, rank: 7 }, to: { file: 1, rank: 4 } })
    const lineA = makeLine('a', [
      shared,
      makeMove({ id: 'a2', side: 'black', from: { file: 7, rank: 2 }, to: { file: 7, rank: 5 } }),
    ])
    const lineB = makeLine('b', [
      shared,
      makeMove({ id: 'b2', side: 'black', from: { file: 0, rank: 3 }, to: { file: 0, rank: 4 } }),
    ])
    const trie = buildMoveTrie([lineA, lineB])
    expect(trie.startGroups.size).toBe(1)
    const [, group] = [...trie.startGroups.entries()][0]
    expect(group.root.lineIds.has('a')).toBe(true)
    expect(group.root.lineIds.has('b')).toBe(true)
    const serialized = serializeTrie(trie.startGroups)
    expect(serialized[0].lineIds).toEqual(expect.arrayContaining(['a', 'b']))
  })

  it('buildMoveTrie separates lines with different initial FEN', () => {
    const lineStandard: LessonLine = {
      id: 'std',
      title: 'Standard',
      moves: [makeMove({ id: 'm1', from: { file: 1, rank: 7 }, to: { file: 1, rank: 4 } })],
    }
    const lineCustom: LessonLine = {
      id: 'custom',
      title: 'Custom FEN',
      initialFen: '4k4/9/9/9/9/9/9/9/9/4K4 b - - 0 1',
      moves: [makeMove({ id: 'm2', side: 'black', from: { file: 4, rank: 0 }, to: { file: 4, rank: 1 } })],
    }
    const trie = buildMoveTrie([lineStandard, lineCustom])
    expect(trie.startGroups.size).toBe(2)
  })

  it('buildActiveNodeId returns start node when no moves played', () => {
    const lineA = makeLine('a', [
      makeMove({ id: 'a1', from: { file: 1, rank: 7 }, to: { file: 1, rank: 4 } }),
    ])
    expect(buildActiveNodeId([lineA], 'a', 0)).toBe(startNodeId('standard'))
  })

  it('buildActiveNodeId walks the path and includes each move signature', () => {
    const lineA = makeLine('a', [
      makeMove({ id: 'a1', from: { file: 1, rank: 7 }, to: { file: 1, rank: 4 } }),
      makeMove({ id: 'a2', side: 'black', from: { file: 7, rank: 2 }, to: { file: 7, rank: 5 } }),
    ])
    const id = buildActiveNodeId([lineA], 'a', 2)
    expect(id.startsWith('start:standard/move:')).toBe(true)
    expect(id.split('/move:')).toHaveLength(3)
  })

  it('formatParsedMoveNotation falls back to coordinates when board is null', () => {
    const move = makeMove({ id: 'x', from: { file: 0, rank: 5 }, to: { file: 0, rank: 4 } })
    const result = formatParsedMoveNotation(move, null)
    expect(result).toMatch(/\(0,5\)→\(0,4\)/)
  })
})

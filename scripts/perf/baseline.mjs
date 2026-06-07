#!/usr/bin/env node
import { readFile } from 'node:fs/promises'
import { performance } from 'node:perf_hooks'

const defaultXiangqiFEN = 'rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1'
const apiBaseUrl = process.env.PERF_API_URL

function byteLength(value) {
  return Buffer.byteLength(typeof value === 'string' ? value : JSON.stringify(value))
}

async function readJSON(path) {
  const readStart = performance.now()
  const body = await readFile(path, 'utf8')
  const parseStart = performance.now()
  const data = JSON.parse(body)
  const done = performance.now()

  return {
    data,
    bytes: byteLength(body),
    readMs: doneNumber(parseStart - readStart),
    parseMs: doneNumber(done - parseStart),
  }
}

function doneNumber(value) {
  return Number(value.toFixed(2))
}

function moveSignature(move) {
  return `${move.side}:${move.from.file},${move.from.rank}:${move.to.file},${move.to.rank}`
}

function phaseFromFen(fen) {
  const placement = String(fen || defaultXiangqiFEN).trim().split(/\s+/)[0]
  const pieceCount = placement ? [...placement].filter((symbol) => /[A-Za-z]/.test(symbol)).length : 32

  return {
    pieceCount,
    phase: pieceCount >= 28 ? 'opening' : pieceCount >= 14 ? 'middlegame' : 'endgame',
  }
}

function lineStartFen(lesson, line) {
  return (line.initialFen || lesson.initialFen || defaultXiangqiFEN).trim()
}

function summarizeLessonData(label, lessonInput) {
  const lessons = Array.isArray(lessonInput) ? lessonInput : [lessonInput]
  const phases = { opening: 0, middlegame: 0, endgame: 0 }
  let lineCount = 0
  let moveCount = 0
  let maxMoves = 0
  let lineLevelInitialFen = 0

  for (const lesson of lessons) {
    for (const line of lesson.lines || []) {
      const moves = line.moves || []
      const { phase } = phaseFromFen(lineStartFen(lesson, line))
      lineCount += 1
      moveCount += moves.length
      maxMoves = Math.max(maxMoves, moves.length)
      phases[phase] += 1
      if (line.initialFen) lineLevelInitialFen += 1
    }
  }

  return {
    label,
    lessons: lessons.length,
    lineCount,
    moveCount,
    avgMoves: doneNumber(moveCount / Math.max(1, lineCount)),
    maxMoves,
    lineLevelInitialFen,
    phases,
  }
}

function combinedOverview(lesson) {
  return {
    id: lesson.id,
    title: lesson.title,
    category: lesson.category,
    difficulty: lesson.difficulty,
    initialFen: lesson.initialFen || defaultXiangqiFEN,
    lines: (lesson.lines || []).map((line) => {
      const { phase, pieceCount } = phaseFromFen(lineStartFen(lesson, line))
      return {
        id: line.id,
        title: line.title,
        initialFen: line.initialFen,
        phase,
        pieceCount,
        moveCount: (line.moves || []).length,
      }
    }),
    choice: lesson.choice,
  }
}

function nextStepWindow(lesson, phase, prefix = [], limit = 1, startFen = '') {
  const from = prefix.length
  const lines = []

  for (const line of lesson.lines || []) {
    const effectiveStartFen = lineStartFen(lesson, line)
    const { phase: linePhase, pieceCount } = phaseFromFen(lineStartFen(lesson, line))
    const moves = line.moves || []
    if (phase && phase !== linePhase) continue
    if (startFen && effectiveStartFen !== startFen) continue
    if (prefix.length > moves.length) continue
    if (!prefix.every((move, index) => moveSignature(moves[index]) === moveSignature(move))) continue

    lines.push({
      lineId: line.id,
      phase: linePhase,
      pieceCount,
      from,
      moves: moves.slice(from, from + limit),
      totalMoves: moves.length,
    })
  }

  return { from, limit, lines }
}

function graphCase(label, lesson, lines, loadedMoveCountForLine) {
  const start = performance.now()
  const startGroups = new Map()
  let replayedMoves = 0
  let nodeCount = 0
  let linkCount = 0

  for (const line of lines) {
    const startKey = lineStartFen(lesson, line) === defaultXiangqiFEN ? 'standard' : lineStartFen(lesson, line)
    let root = startGroups.get(startKey)
    if (!root) {
      root = { children: new Map(), lineIds: new Set() }
      startGroups.set(startKey, root)
      nodeCount += 1
    }
    root.lineIds.add(line.id)

    let parent = root
    const moves = (line.moves || []).slice(0, loadedMoveCountForLine(line))
    for (const move of moves) {
      replayedMoves += 1
      const signature = moveSignature(move)
      let child = parent.children.get(signature)
      if (!child) {
        child = { children: new Map(), lineIds: new Set() }
        parent.children.set(signature, child)
        nodeCount += 1
        linkCount += 1
      }
      child.lineIds.add(line.id)
      parent = child
    }
  }

  return {
    label,
    lines: lines.length,
    startGroups: startGroups.size,
    nodes: nodeCount,
    links: linkCount,
    replayedMoves,
    buildMs: doneNumber(performance.now() - start),
  }
}

function firstMoveMatches(line, move) {
  const firstMove = (line.moves || [])[0]
  return Boolean(firstMove && move && moveSignature(firstMove) === moveSignature(move))
}

async function measureEndpoint(path, init) {
  if (!apiBaseUrl) return null

  const start = performance.now()
  const response = await fetch(`${apiBaseUrl}${path}`, init)
  const body = await response.text()

  return {
    path,
    status: response.status,
    etag: response.headers.get('etag') || undefined,
    cache: response.headers.get('x-cache') || undefined,
    dataVersion: response.headers.get('x-data-version') || undefined,
    bytes: byteLength(body),
    timeMs: doneNumber(performance.now() - start),
  }
}

async function measureConditionalEndpoint(path) {
  const first = await measureEndpoint(path)
  if (!first?.etag) return { first }

  return {
    first,
    revalidated: await measureEndpoint(path, {
      headers: {
        'If-None-Match': first.etag,
      },
    }),
  }
}

async function measureRepeatedEndpoint(path, init) {
  return {
    first: await measureEndpoint(path, init),
    repeated: await measureEndpoint(path, init),
  }
}

const [{ data: lessons, ...catalogRead }, { data: combinedLesson, ...combinedRead }] = await Promise.all([
  readJSON('apps/api/data/lessons/lessons.json'),
  readJSON('apps/api/data/lessons/combined-lesson.json'),
])

const overview = combinedOverview(combinedLesson)
const rootOpeningWindow = nextStepWindow(combinedLesson, 'opening', [], 1, defaultXiangqiFEN)
const topOpeningMove = rootOpeningWindow.lines
  .flatMap((line) => line.moves)
  .reduce((counts, move) => {
    const key = moveSignature(move)
    counts.set(key, { move, count: (counts.get(key)?.count || 0) + 1 })
    return counts
  }, new Map())
const topPrefix = [...topOpeningMove.values()].sort((left, right) => right.count - left.count)[0]?.move
const branchWindow = topPrefix ? nextStepWindow(combinedLesson, 'opening', [topPrefix], 1, defaultXiangqiFEN) : { lines: [] }
const openingLines = overview.lines
  .filter((line) => line.phase === 'opening')
  .map((summary) => combinedLesson.lines.find((line) => line.id === summary.id))
  .filter(Boolean)

const output = {
  generatedAt: new Date().toISOString(),
  data: {
    catalogRead,
    combinedRead,
    catalog: summarizeLessonData('catalog', lessons),
    combined: summarizeLessonData('combined', combinedLesson),
    combinedOverviewBytes: byteLength(overview),
    rootOpeningNextStepBytes: byteLength(rootOpeningWindow),
    rootOpeningNextStepLines: rootOpeningWindow.lines.length,
    branchNextStepBytes: byteLength(branchWindow),
    branchNextStepLines: branchWindow.lines.length,
  },
  graph: [
    graphCase('opening-root-loaded', combinedLesson, openingLines, (line) => (lineStartFen(combinedLesson, line) === defaultXiangqiFEN ? 1 : 0)),
    graphCase('opening-one-branch-depth-2', combinedLesson, openingLines, (line) =>
      lineStartFen(combinedLesson, line) === defaultXiangqiFEN && firstMoveMatches(line, topPrefix)
        ? 2
        : lineStartFen(combinedLesson, line) === defaultXiangqiFEN
          ? 1
          : 0,
    ),
    graphCase('opening-full-phase-loaded', combinedLesson, openingLines, (line) => (line.moves || []).length),
  ],
  api: apiBaseUrl
    ? {
        health: await measureEndpoint('/api/health'),
        categories: await measureConditionalEndpoint('/api/categories'),
        lessons: await measureConditionalEndpoint('/api/lessons'),
        combinedOverview: await measureConditionalEndpoint('/api/combined-lesson'),
        combinedNextSteps: await measureRepeatedEndpoint('/api/combined-lesson/next-steps', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            phase: 'opening',
            initialFen: defaultXiangqiFEN,
            from: 0,
            limit: 1,
            prefix: [],
          }),
        }),
      }
    : 'Set PERF_API_URL to include live API endpoint timings.',
}

console.log(JSON.stringify(output, null, 2))

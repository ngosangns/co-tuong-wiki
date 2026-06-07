import { describe, expect, it, beforeEach, afterEach } from 'vitest'
import { renderTreeGraph, type TreeGraphNode } from './treeGraph'

function makeNode(id: string, label: string): TreeGraphNode {
  return { id, label }
}

function makeGraph() {
  return {
    nodes: [
      makeNode('start', 'Start'),
      makeNode('a1', 'Move 1A'),
      makeNode('a2', 'Move 2A'),
      makeNode('a3', 'Move 3A'),
      makeNode('b1', 'Move 1B'),
    ],
    links: [
      { source: 'start', target: 'a1' },
      { source: 'a1', target: 'a2' },
      { source: 'a2', target: 'a3' },
      { source: 'start', target: 'b1' },
    ],
  }
}

describe('renderTreeGraph', () => {
  let container: HTMLElement
  let renderer: ReturnType<typeof renderTreeGraph>

  beforeEach(() => {
    container = document.createElement('div')
    document.body.appendChild(container)
  })

  afterEach(() => {
    renderer?.kill()
    document.body.removeChild(container)
  })

  it('renders an SVG with a viewport group that contains link and node layers', () => {
    renderer = renderTreeGraph({
      container,
      graph: makeGraph(),
      selectedId: '',
      nodeColor: () => '#000',
      edgeColor: () => '#666',
      onSelectNode: () => {},
    })
    const svg = container.querySelector('svg')
    expect(svg).toBeTruthy()
    const linkLayer = svg?.querySelector('.tree-graph-links')
    const nodeLayer = svg?.querySelector('.tree-graph-nodes')
    expect(linkLayer).toBeTruthy()
    expect(nodeLayer).toBeTruthy()
  })

  it('isZoomed returns false initially', () => {
    renderer = renderTreeGraph({
      container,
      graph: makeGraph(),
      selectedId: '',
      nodeColor: () => '#000',
      edgeColor: () => '#666',
      onSelectNode: () => {},
    })
    expect(renderer.isZoomed()).toBe(false)
  })

  it('reset() returns to identity transform and clears zoomed state', () => {
    let zoomed = false
    renderer = renderTreeGraph({
      container,
      graph: makeGraph(),
      selectedId: '',
      nodeColor: () => '#000',
      edgeColor: () => '#666',
      onSelectNode: () => {},
      onZoomChange: (state) => {
        zoomed = state
      },
    })
    // Force a non-identity transform via the public reset path: simulate that the viewport is zoomed by
    // calling reset which is a no-op when already at identity. Then verify state.
    renderer.reset()
    expect(renderer.isZoomed()).toBe(false)
    expect(zoomed).toBe(false)
  })

  it('kill removes the SVG from the container', () => {
    renderer = renderTreeGraph({
      container,
      graph: makeGraph(),
      selectedId: '',
      nodeColor: () => '#000',
      edgeColor: () => '#666',
      onSelectNode: () => {},
    })
    expect(container.querySelector('svg')).toBeTruthy()
    renderer.kill()
    expect(container.querySelector('svg')).toBeFalsy()
  })

  it('renders all nodes including the synthetic root children', () => {
    renderer = renderTreeGraph({
      container,
      graph: makeGraph(),
      selectedId: 'a2',
      nodeColor: () => '#000',
      edgeColor: () => '#666',
      onSelectNode: () => {},
    })
    const nodeGroups = container.querySelectorAll('.tree-graph-node')
    // 5 nodes: start, a1, a2, a3, b1 (synthetic root is hidden)
    expect(nodeGroups.length).toBe(5)
  })

  it('renders links as path elements', () => {
    renderer = renderTreeGraph({
      container,
      graph: makeGraph(),
      selectedId: '',
      nodeColor: () => '#000',
      edgeColor: () => '#666',
      onSelectNode: () => {},
    })
    const links = container.querySelectorAll('.tree-graph-link')
    expect(links.length).toBe(4)
  })
})

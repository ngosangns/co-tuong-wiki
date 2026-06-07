import { hierarchy, tree, type HierarchyPointNode } from 'd3-hierarchy'
import { zoom, zoomIdentity, type D3ZoomEvent, type ZoomBehavior } from 'd3-zoom'
import { select } from 'd3-selection'

export interface TreeGraphNode {
  id: string
  label: string
  state?: string
  [key: string]: unknown
}

export interface TreeGraphLink {
  source: string
  target: string
  type?: string
}

export interface TreeGraphData {
  nodes: TreeGraphNode[]
  links: TreeGraphLink[]
}

export interface TreeGraphRenderer {
  fit(): void
  reset(): void
  kill(): void
  setSelected(id: string): void
  setStates(states: Record<string, string>): void
  isZoomed(): boolean
}

interface TreeDatum {
  node: TreeGraphNode
  children: TreeDatum[]
  synthetic?: boolean
}

interface RenderTreeGraphOptions {
  container: HTMLElement
  graph: TreeGraphData
  selectedId: string
  nodeColor: (node: TreeGraphNode) => string
  edgeColor: (type: string) => string
  labelColor?: string
  onSelectNode: (node: TreeGraphNode) => void
  onClearSelection?: () => void
  onZoomChange?: (isZoomed: boolean) => void
}

const syntheticRootId = '__tree_root__'

export function renderTreeGraph(options: RenderTreeGraphOptions): TreeGraphRenderer {
  const normalized = normalizeGraph(options.graph)
  const root = tree<TreeDatum>().nodeSize([72, 210])(hierarchy(normalized.root, (node) => node.children))
  const visibleNodes = root.descendants().filter((node) => !node.data.synthetic)
  const visibleLinks = root.links().filter((link) => !link.source.data.synthetic)
  const related = buildRelatedMap(options.graph.links)
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg')
  const viewport = document.createElementNS('http://www.w3.org/2000/svg', 'g')
  const linkLayer = document.createElementNS('http://www.w3.org/2000/svg', 'g')
  const nodeLayer = document.createElementNS('http://www.w3.org/2000/svg', 'g')
  const nodeElements = new Map<string, SVGGElement>()
  const nodeData = new Map<string, TreeGraphNode>()
  const linkElements = new Map<string, SVGPathElement>()
  let selectedId = options.selectedId && normalized.nodes.has(options.selectedId) ? options.selectedId : ''
  let disposed = false

  svg.setAttribute('class', 'tree-graph-svg')
  svg.setAttribute('role', 'img')
  svg.setAttribute('aria-label', 'Move tree')
  linkLayer.setAttribute('class', 'tree-graph-links')
  nodeLayer.setAttribute('class', 'tree-graph-nodes')
  viewport.append(linkLayer, nodeLayer)
  svg.append(viewport)
  options.container.append(svg)

  for (const link of visibleLinks) {
    const relation = link.target.data.node.edgeType || 'next'
    const path = document.createElementNS('http://www.w3.org/2000/svg', 'path')
    path.setAttribute('class', 'tree-graph-link')
    path.setAttribute('d', linkPath(link.source, link.target))
    path.setAttribute('fill', 'none')
    path.setAttribute('stroke', options.edgeColor(String(relation)))
    path.setAttribute('stroke-width', relation === 'active' ? '2.4' : '1.4')
    path.setAttribute('data-source', link.source.data.node.id)
    path.setAttribute('data-target', link.target.data.node.id)
    linkLayer.append(path)
    linkElements.set(`${link.source.data.node.id}->${link.target.data.node.id}`, path)
  }

  for (const point of visibleNodes) {
    const node = point.data.node
    const group = document.createElementNS('http://www.w3.org/2000/svg', 'g')
    const circle = document.createElementNS('http://www.w3.org/2000/svg', 'circle')
    const text = document.createElementNS('http://www.w3.org/2000/svg', 'text')
    const title = document.createElementNS('http://www.w3.org/2000/svg', 'title')

    group.setAttribute('class', 'tree-graph-node')
    group.setAttribute('transform', `translate(${point.y},${point.x})`)
    group.setAttribute('tabindex', '0')
    group.setAttribute('role', 'button')
    group.setAttribute('aria-label', node.label || node.id)
    circle.setAttribute('r', String(nodeSize(node)))
    circle.setAttribute('fill', options.nodeColor(node))
    text.setAttribute('x', '14')
    text.setAttribute('y', '4')
    text.setAttribute('class', 'tree-graph-label')
    text.setAttribute('fill', options.labelColor || '#f5efe2')
    text.textContent = shortLabel(node.label || node.id)
    title.textContent = node.label || node.id
    group.append(circle, text, title)
    group.addEventListener('click', (event) => {
      event.stopPropagation()
      selectedId = node.id
      updateSelection()
      options.onSelectNode(node)
    })
    group.addEventListener('keydown', (event) => {
      if (event.key !== 'Enter' && event.key !== ' ') return
      event.preventDefault()
      selectedId = node.id
      updateSelection()
      options.onSelectNode(node)
    })
    group.addEventListener('keydown', (event: KeyboardEvent) => {
      if (event.key !== 'ArrowDown' && event.key !== 'ArrowRight') return
      const ids = Array.from(nodeElements.keys())
      const idx = ids.indexOf(node.id)
      if (idx < 0) return
      const next = ids[idx + 1] ?? ids[0]
      if (next === node.id) return
      event.preventDefault()
      const nextElement = nodeElements.get(next)
      if (nextElement instanceof HTMLElement || nextElement instanceof SVGElement) {
        nextElement.focus()
      }
    })
    nodeLayer.append(group)
    nodeElements.set(node.id, group)
    nodeData.set(node.id, node)
  }

  svg.addEventListener('click', () => {
    if (!selectedId) return
    selectedId = ''
    updateSelection()
    options.onClearSelection?.()
  })

  function updateSelection() {
    if (disposed) return
    const selectedRelated = selectedId ? related.get(selectedId) : undefined
    for (const [id, element] of nodeElements) {
      const isSelected = id === selectedId
      const isRelated = Boolean(selectedRelated?.has(id))
      element.classList.toggle('is-selected', isSelected)
      element.classList.toggle('is-dimmed', Boolean(selectedId) && !isSelected && !isRelated)
    }
    for (const [key, element] of linkElements) {
      const [source, target] = key.split('->')
      const isRelated = Boolean(selectedId) && (source === selectedId || target === selectedId)
      element.classList.toggle('is-dimmed', Boolean(selectedId) && !isRelated)
    }
  }

  function followSelectedNode(behavior: ScrollBehavior = 'smooth') {
    if (disposed || !selectedId) return
    nodeElements.get(selectedId)?.scrollIntoView({
      behavior,
      block: 'center',
      inline: 'center',
    })
  }

  function fit() {
    if (disposed) return
    const bounds = layoutBounds(visibleNodes)
    svg.style.width = `${bounds.width}px`
    svg.style.height = `${bounds.height}px`
    svg.setAttribute('viewBox', `${bounds.minY} ${bounds.minX} ${bounds.width} ${bounds.height}`)
    zoomBehavior.transform(selection, zoomIdentity)
  }

  function applyZoom(transform: { k: number; x: number; y: number }) {
    if (disposed) return
    viewport.setAttribute('transform', `translate(${transform.x}, ${transform.y}) scale(${transform.k})`)
  }

  function isZoomed() {
    return currentTransform.k !== 1 || currentTransform.x !== 0 || currentTransform.y !== 0
  }

  function reset() {
    if (disposed) return
    zoomBehavior.transform(selection, zoomIdentity)
  }

  const selection = select(svg)
  const currentTransform = { k: 1, x: 0, y: 0 }
  const zoomBehavior: ZoomBehavior<SVGSVGElement, unknown> = zoom<SVGSVGElement, unknown>()
    .scaleExtent([0.25, 4])
    .filter((event: Event) => {
      if (event.type === 'wheel') return true
      if (event.type === 'dblclick') return false
      const target = event.target as Element | null
      if (!target) return false
      return !target.closest('.tree-graph-node')
    })
    .on('zoom', (event: D3ZoomEvent<SVGSVGElement, unknown>) => {
      const t = event.transform
      currentTransform.k = t.k
      currentTransform.x = t.x
      currentTransform.y = t.y
      applyZoom(t)
      options.onZoomChange?.(t.k !== 1 || t.x !== 0 || t.y !== 0)
    })
  selection.call(zoomBehavior)
  selection.on('dblclick.zoom', null)
  selection.on('dblclick', () => {
    zoomBehavior.transform(selection, zoomIdentity)
  })

  function setStates(states: Record<string, string>) {
    if (disposed) return
    for (const [id, state] of Object.entries(states)) {
      const node = nodeData.get(id)
      const element = nodeElements.get(id)
      const circle = element?.querySelector('circle')
      if (!node || !circle) continue
      node.state = state
      circle.setAttribute('fill', options.nodeColor(node))
    }
    for (const [key, element] of linkElements) {
      const [, target] = key.split('->')
      const targetState = nodeData.get(target)?.state
      const relation = targetState === 'active' || targetState === 'past' ? 'active' : 'next'
      element.setAttribute('stroke', options.edgeColor(relation))
      element.setAttribute('stroke-width', relation === 'active' ? '2.4' : '1.4')
    }
  }

  fit()
  updateSelection()
  requestAnimationFrame(() => followSelectedNode('auto'))

  return {
    fit,
    reset,
    kill: () => {
      disposed = true
      zoomBehavior.on('zoom', null)
      selection.on('.zoom', null)
      svg.remove()
    },
    setSelected: (id: string) => {
      selectedId = id && normalized.nodes.has(id) ? id : ''
      updateSelection()
      followSelectedNode()
    },
    setStates,
    isZoomed,
  }
}

function normalizeGraph(graph: TreeGraphData) {
  const nodes = new Map<string, TreeGraphNode>()
  const childrenBySource = new Map<string, TreeGraphLink[]>()
  const targets = new Set<string>()

  for (const node of graph.nodes) {
    const id = String(node.id || '').trim()
    if (!id || nodes.has(id)) continue
    nodes.set(id, { ...node, id, label: node.label || id })
  }
  for (const link of graph.links) {
    if (!nodes.has(link.source) || !nodes.has(link.target) || link.source === link.target) continue
    const links = childrenBySource.get(link.source) ?? []
    links.push(link)
    childrenBySource.set(link.source, links)
    targets.add(link.target)
  }

  const roots = [...nodes.values()].filter((node) => !targets.has(node.id))
  const seen = new Set<string>()
  const buildDatum = (node: TreeGraphNode): TreeDatum => {
    if (seen.has(node.id)) return { node, children: [] }
    seen.add(node.id)
    const children = (childrenBySource.get(node.id) ?? [])
      .map((link) => {
        const child = nodes.get(link.target)
        return child ? buildDatum({ ...child, edgeType: link.type || 'next' }) : null
      })
      .filter((child): child is TreeDatum => Boolean(child))
    return { node, children }
  }

  // d3.tree expects a single root, so independent lesson lines hang under a hidden synthetic root.
  const root: TreeDatum = {
    node: { id: syntheticRootId, label: '' },
    children: roots.map(buildDatum),
    synthetic: true,
  }

  return { root, nodes }
}

function buildRelatedMap(links: TreeGraphLink[]) {
  const related = new Map<string, Set<string>>()
  for (const link of links) {
    const source = related.get(link.source) ?? new Set<string>()
    const target = related.get(link.target) ?? new Set<string>()
    source.add(link.target)
    target.add(link.source)
    related.set(link.source, source)
    related.set(link.target, target)
  }
  return related
}

function linkPath(source: HierarchyPointNode<TreeDatum>, target: HierarchyPointNode<TreeDatum>) {
  const midY = (source.y + target.y) / 2
  return `M${source.y},${source.x} C${midY},${source.x} ${midY},${target.x} ${target.y},${target.x}`
}

function layoutBounds(nodes: HierarchyPointNode<TreeDatum>[]) {
  if (!nodes.length) return { minX: -40, minY: -40, width: 80, height: 80 }
  const xs = nodes.map((node) => node.x)
  const ys = nodes.map((node) => node.y)
  const minX = Math.min(...xs) - 48
  const maxX = Math.max(...xs) + 48
  const minY = Math.min(...ys) - 48
  const maxY = Math.max(...ys) + 240
  return {
    minX,
    minY,
    width: Math.max(1, maxY - minY),
    height: Math.max(1, maxX - minX),
  }
}

function nodeSize(node: TreeGraphNode) {
  if (node.state === 'active') return 8
  if (node.state === 'past') return 7
  return 6
}

function shortLabel(label: string) {
  return label.length > 34 ? `${label.slice(0, 31)}...` : label
}

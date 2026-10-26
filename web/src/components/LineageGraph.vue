<script setup lang="ts">
import { computed } from 'vue'
import type { GraphNode } from '@/api/types'
import { branchTone } from '@/utils/format'

const props = defineProps<{ nodes: GraphNode[] }>()
const emit = defineEmits<{ (e: 'select', id: number): void }>()

const GUTTER = 172
const PLOT_W = 848
const LANE_H = 54
const TOP = 26
const BOTTOM = 44
const W = GUTTER + PLOT_W + 20

interface Lane {
  node: GraphNode
  lane: number
  start: number
  end: number
  color: string
}

const byId = computed(() => new Map(props.nodes.map((n) => [n.id, n])))

function depthOf(node: GraphNode): number {
  let depth = 0
  let cur: GraphNode | undefined = node
  const seen = new Set<number>()
  while (cur?.parentId && !seen.has(cur.id)) {
    seen.add(cur.id)
    cur = byId.value.get(cur.parentId)
    if (cur) depth++
  }
  return depth
}

const ordered = computed(() =>
  [...props.nodes].sort(
    (a, b) => depthOf(a) - depthOf(b) || (a.forkYear || 0) - (b.forkYear || 0) || a.id - b.id,
  ),
)

const years = computed(() => {
  if (!props.nodes.length) return { min: 1949, max: 2026 }
  const starts = props.nodes.map((n) => (n.parentId ? n.forkYear || n.startYear : n.startYear) || 1949)
  const ends = props.nodes.map((n) => n.endYear || n.startYear || n.forkYear || 1949)
  const min = Math.min(...starts)
  const max = Math.max(...ends)
  return { min: min - 1, max: Math.max(max + 1, min + 6) }
})

const lanes = computed<Lane[]>(() =>
  ordered.value.map((node, index) => {
    const start = (node.parentId ? node.forkYear || node.startYear : node.startYear) || years.value.min
    const end = Math.max(node.endYear || start + 1, start + 1)
    return { node, lane: index, start, end, color: branchTone(node.official, node.id) }
  }),
)

const height = computed(() => TOP + lanes.value.length * LANE_H + BOTTOM)

function x(year: number): number {
  const { min, max } = years.value
  return GUTTER + ((year - min) / (max - min)) * PLOT_W
}

function y(lane: number): number {
  return TOP + lane * LANE_H + LANE_H / 2
}

const ticks = computed(() => {
  const list: number[] = []
  const start = Math.ceil(years.value.min / 5) * 5
  for (let y = start; y <= years.value.max; y += 5) list.push(y)
  return list
})

const laneById = computed(() => new Map(lanes.value.map((l) => [l.node.id, l])))

interface ForkLink {
  key: number
  d: string
  color: string
  parentX: number
  parentY: number
  childX: number
  childY: number
}

const forks = computed<ForkLink[]>(() => {
  const out: ForkLink[] = []
  for (const lane of lanes.value) {
    const { node } = lane
    if (!node.parentId) continue
    const parent = laneById.value.get(node.parentId)
    if (!parent) continue
    const px = x(Math.min(Math.max(node.forkYear || lane.start, years.value.min), years.value.max))
    const py = y(parent.lane)
    const cx = x(lane.start)
    const cy = y(lane.lane)
    const mid = Math.max(px + 18, (px + cx) / 2)
    out.push({
      key: node.id,
      d: `M ${px} ${py} C ${mid} ${py}, ${mid} ${cy}, ${cx} ${cy}`,
      color: lane.color,
      parentX: px,
      parentY: py,
      childX: cx,
      childY: cy,
    })
  }
  return out
})

function shortName(name: string): string {
  const clean = name.replace(/^(野史|别史|正史)·/, '')
  return clean.length > 9 ? `${clean.slice(0, 9)}…` : clean
}

function tooltip(node: GraphNode): string {
  const span = node.startYear ? `${node.startYear}–${node.endYear || node.startYear}` : '年份未详'
  return `${node.name}｜${span}｜${node.eventCount} 条实录｜${node.forkCount} 处分叉｜主笔 ${node.owner || '史馆'}`
}
</script>

<template>
  <div class="graph-panel">
    <div class="graph-legend">
      <span style="color: var(--ink)"><i class="k"></i>官修</span>
      <span style="color: var(--ochre)"><i class="k"></i>别史</span>
      <span style="color: var(--cinnabar)"><i class="k"></i>野史</span>
      <span style="margin-left: auto">横轴为年份，纵轴为史枝 · 点击可入</span>
    </div>
    <svg class="graph-svg" :viewBox="`0 0 ${W} ${height}`" :style="{ minHeight: `${Math.min(height, 470)}px` }">
      <g>
        <line
          v-for="t in ticks"
          :key="`grid-${t}`"
          :x1="x(t)"
          :x2="x(t)"
          :y1="TOP - 10"
          :y2="height - BOTTOM + 4"
          stroke="var(--rule-2)"
          stroke-width="0.7"
          stroke-dasharray="1 4"
        />
        <text
          v-for="t in ticks"
          :key="`tick-${t}`"
          :x="x(t)"
          :y="height - BOTTOM + 24"
          text-anchor="middle"
          class="axis-label"
        >
          {{ t }}
        </text>
      </g>

      <g v-for="fork in forks" :key="`fork-${fork.key}`">
        <path :d="fork.d" fill="none" :stroke="fork.color" stroke-width="1" stroke-opacity="0.55" />
        <circle :cx="fork.parentX" :cy="fork.parentY" r="2.4" :fill="fork.color" />
      </g>

      <g v-for="lane in lanes" :key="lane.node.id" class="branch-group">
        <title>{{ tooltip(lane.node) }}</title>
        <text :x="GUTTER - 16" :y="y(lane.lane) + 4" text-anchor="end" class="branch-label">
          {{ shortName(lane.node.name) }}
        </text>
        <text :x="GUTTER - 16" :y="y(lane.lane) + 18" text-anchor="end" class="branch-count">
          {{ lane.start }}–{{ lane.end }} · {{ lane.node.eventCount }} 条
        </text>
        <line
          class="branch-line"
          :x1="x(lane.start)"
          :y1="y(lane.lane)"
          :x2="x(lane.end)"
          :y2="y(lane.lane)"
          :stroke="lane.color"
          :stroke-width="lane.node.official ? 2.4 : 1.5"
          stroke-linecap="round"
        />
        <circle :cx="x(lane.start)" :cy="y(lane.lane)" r="4" :fill="lane.color" />
        <circle
          v-if="lane.node.official"
          :cx="x(lane.start)"
          :cy="y(lane.lane)"
          r="6.5"
          fill="none"
          :stroke="lane.color"
          stroke-width="0.8"
          stroke-opacity="0.55"
        />
        <rect
          class="branch-hit"
          :x="x(lane.start)"
          :y="y(lane.lane) - 12"
          :width="Math.max(x(lane.end) - x(lane.start), 10)"
          height="24"
          @click="emit('select', lane.node.id)"
        >
          <title>{{ tooltip(lane.node) }}</title>
        </rect>
      </g>
    </svg>
    <div v-if="!nodes.length" class="empty">谱系尚未展开</div>
  </div>
</template>

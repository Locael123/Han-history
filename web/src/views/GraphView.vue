<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '@/api'
import type { GraphNode } from '@/api/types'
import LineageGraph from '@/components/LineageGraph.vue'
import { branchTone } from '@/utils/format'

const router = useRouter()
const nodes = ref<GraphNode[]>([])
const loading = ref(true)

const ordered = computed(() =>
  [...nodes.value].sort((a, b) => (a.forkYear || a.startYear || 0) - (b.forkYear || b.startYear || 0) || a.id - b.id),
)

onMounted(async () => {
  try {
    nodes.value = await api.graph()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
})

function open(id: number) {
  router.push(`/branches/${id}`)
}
</script>

<template>
  <div class="page-wrap" v-loading="loading">
    <header class="page-head">
      <div class="kicker">谱系 · 第二辑</div>
      <h2>一树千枝</h2>
      <p class="lede">
        横轴是年份，纵轴是史枝。线自何处起，便自何处分。官修主干以墨色、双圈标出；野史各枝，各有其色。
        点任意一线，可入该枝。
      </p>
    </header>

    <div class="graph-frame">
      <LineageGraph :nodes="nodes" @select="open" />
    </div>

    <section class="section">
      <div class="section-head">
        <span class="ordinal">附</span>
        <h2>谱系表</h2>
        <span class="rule"></span>
        <span class="muted">{{ nodes.length }} 枝</span>
      </div>
      <div class="lineage-table">
        <div class="lineage-row head">
          <span>史枝</span>
          <span>所自出</span>
          <span class="num">分叉年</span>
          <span class="num">实录</span>
          <span class="num">分叉</span>
          <span>主笔</span>
        </div>
        <RouterLink
          v-for="n in ordered"
          :key="n.id"
          :to="`/branches/${n.id}`"
          class="lineage-row"
          :style="{ '--branch-color': branchTone(n.official, n.id) }"
        >
          <span class="name">
            <i class="tick"></i>
            {{ n.name }}
            <em v-if="n.official">官修</em>
          </span>
          <span class="dim">{{ n.parentId ? '分叉' : '主干' }}</span>
          <span class="num">{{ n.parentId ? n.forkYear || '—' : '—' }}</span>
          <span class="num">{{ n.eventCount }}</span>
          <span class="num">{{ n.forkCount }}</span>
          <span class="dim">{{ n.owner || '史馆' }}</span>
        </RouterLink>
      </div>
    </section>
  </div>
</template>

<style scoped>
.page-head {
  max-width: 720px;
  padding: 20px 0 40px;
}

.page-head h2 {
  font-family: var(--serif);
  font-size: clamp(34px, 4.6vw, 52px);
  font-weight: 500;
  letter-spacing: 0.1em;
  margin: 12px 0 18px;
}

.graph-frame {
  border-top: 1px solid var(--ink);
  border-bottom: 1px solid var(--ink);
  padding: 10px 0;
}

.lineage-table {
  border-top: 1px solid var(--ink);
}

.lineage-row {
  display: grid;
  grid-template-columns: minmax(200px, 2fr) 1fr 90px 90px 90px 1fr;
  align-items: baseline;
  gap: 16px;
  padding: 14px 10px;
  border-bottom: 1px solid var(--rule-2);
  color: var(--ink-2);
  font-size: 13.5px;
}

.lineage-row.head {
  font-size: 10.5px;
  letter-spacing: 0.16em;
  color: var(--ink-3);
  border-bottom: 1px solid var(--rule);
}

.lineage-row:not(.head):hover {
  background: var(--wash);
}

.lineage-row .name {
  font-family: var(--serif);
  font-size: 16px;
  color: var(--ink);
  display: flex;
  align-items: baseline;
  gap: 10px;
}

.lineage-row .tick {
  width: 3px;
  height: 14px;
  background: var(--branch-color);
  display: inline-block;
  flex: none;
  transform: translateY(1px);
}

.lineage-row .name em {
  font-style: normal;
  font-size: 10px;
  letter-spacing: 0.12em;
  border: 1px solid var(--ink-3);
  color: var(--ink-3);
  padding: 1px 4px;
}

.num {
  text-align: right;
  font-family: var(--mono);
  font-size: 12px;
}

.dim {
  color: var(--ink-3);
  font-size: 12px;
}

@media (max-width: 860px) {
  .lineage-row {
    grid-template-columns: 1.6fr 1fr 70px 60px;
  }

  .lineage-row > :nth-child(5),
  .lineage-row > :nth-child(6) {
    display: none;
  }
}
</style>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '@/api'
import type { BranchCard } from '@/api/types'
import BranchCardView from '@/components/BranchCard.vue'

const route = useRoute()
const router = useRouter()

const list = ref<BranchCard[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 9
const scope = ref((route.query.scope as string) || 'all')
const sort = ref((route.query.sort as string) || 'hot')
const keyword = ref((route.query.q as string) || '')
const loading = ref(true)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const scopes = [
  { key: 'all', label: '全部' },
  { key: 'official', label: '官修' },
  { key: 'folk', label: '野史' },
]
const sorts = [
  { key: 'hot', label: '最热' },
  { key: 'new', label: '新立' },
  { key: 'forks', label: '多分叉' },
  { key: 'events', label: '条目多' },
]

async function load() {
  loading.value = true
  try {
    const res = await api.branches({
      scope: scope.value,
      sort: sort.value,
      q: keyword.value.trim(),
      page: page.value,
      pageSize,
    })
    list.value = res.list
    total.value = res.total
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

function syncQuery() {
  router.replace({
    query: {
      ...(scope.value !== 'all' ? { scope: scope.value } : {}),
      ...(sort.value !== 'hot' ? { sort: sort.value } : {}),
      ...(keyword.value.trim() ? { q: keyword.value.trim() } : {}),
    },
  })
}

function switchScope(key: string) {
  scope.value = key
  page.value = 1
  syncQuery()
  load()
}

function switchSort(key: string) {
  sort.value = key
  page.value = 1
  syncQuery()
  load()
}

function search() {
  page.value = 1
  syncQuery()
  load()
}

function goPage(n: number) {
  if (n < 1 || n > totalPages.value) return
  page.value = n
  load()
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

watch(
  () => route.query,
  (q) => {
    const nextScope = (q.scope as string) || 'all'
    const nextSort = (q.sort as string) || 'hot'
    const nextQ = (q.q as string) || ''
    if (nextScope !== scope.value || nextSort !== sort.value || nextQ !== keyword.value) {
      scope.value = nextScope
      sort.value = nextSort
      keyword.value = nextQ
      page.value = 1
      load()
    }
  },
)

onMounted(load)
</script>

<template>
  <div class="page-wrap" v-loading="loading">
    <header class="page-head">
      <div class="kicker">史枝 · 总目</div>
      <h2>史枝总目</h2>
      <p class="lede">
        一条主干，数条别史，若干野史。每一枝都从前枝的某一年分出，带走此前所有条目，再从那一处往下写。
      </p>
    </header>

    <div class="filterbar">
      <span class="label">体例</span>
      <div class="seg">
        <button v-for="s in scopes" :key="s.key" :class="{ on: scope === s.key }" @click="switchScope(s.key)">
          {{ s.label }}
        </button>
      </div>
      <span class="label">次序</span>
      <div class="seg">
        <button v-for="s in sorts" :key="s.key" :class="{ on: sort === s.key }" @click="switchSort(s.key)">
          {{ s.label }}
        </button>
      </div>
      <span class="spacer"></span>
      <form class="searchbox" @submit.prevent="search">
        <input v-model="keyword" type="search" placeholder="搜枝名、小引、枝主" aria-label="搜索史枝" />
        <button type="submit" class="text-link">检</button>
      </form>
    </div>

    <div v-if="list.length" class="branch-grid">
      <BranchCardView v-for="b in list" :key="b.id" :branch="b" />
    </div>
    <div v-else-if="!loading" class="empty">
      此间尚无史枝。或换一个说法再检。
    </div>

    <div v-if="totalPages > 1" class="pager">
      <button class="btn btn-sm btn-ghost" :disabled="page <= 1" @click="goPage(page - 1)">上一页</button>
      <span class="pager-num">第 {{ page }} / {{ totalPages }} 页　共 {{ total }} 枝</span>
      <button class="btn btn-sm btn-ghost" :disabled="page >= totalPages" @click="goPage(page + 1)">下一页</button>
    </div>

    <div class="branch-cta">
      <div>
        <div class="kicker">立一枝</div>
        <p>分叉要选一处年份。先去任意一枝，在你想接续的那条实录旁点「自此分叉」。</p>
      </div>
      <RouterLink class="btn btn-cinnabar" to="/graph">去谱系选一枝</RouterLink>
    </div>
  </div>
</template>

<style scoped>
.page-head {
  max-width: 720px;
  padding: 20px 0 34px;
}

.page-head h2 {
  font-family: var(--serif);
  font-size: clamp(34px, 4.6vw, 52px);
  font-weight: 500;
  letter-spacing: 0.1em;
  margin: 12px 0 16px;
}

.pager {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 18px;
  margin-top: 36px;
}

.pager-num {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--ink-3);
  letter-spacing: 0.08em;
}

.branch-cta {
  margin-top: 56px;
  border-top: 1px solid var(--ink);
  padding-top: 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.branch-cta p {
  margin: 8px 0 0;
  font-size: 13px;
  color: var(--ink-3);
}

@media (max-width: 720px) {
  .branch-cta {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>

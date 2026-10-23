<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '@/api'
import type { TimelineItem } from '@/api/types'
import { shortDate } from '@/utils/format'

const route = useRoute()
const router = useRouter()

const list = ref<TimelineItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 15
const scope = ref((route.query.scope as string) || 'all')
const sort = ref((route.query.sort as string) || 'year')
const keyword = ref((route.query.q as string) || '')
const loading = ref(true)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const scopes = [
  { key: 'all', label: '全部' },
  { key: 'official', label: '官修' },
  { key: 'folk', label: '野史' },
]
const sorts = [
  { key: 'year', label: '年代' },
  { key: 'latest', label: '最新录入' },
  { key: 'hot', label: '最多赞' },
]

async function load() {
  loading.value = true
  try {
    const res = await api.events({
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
      ...(sort.value !== 'year' ? { sort: sort.value } : {}),
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
    const nextSort = (q.sort as string) || 'year'
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
      <div class="kicker">实录 · 全站</div>
      <h2>全站实录</h2>
      <p class="lede">
        各枝条目，汇于此。官修与野史并列，只按年份与录入先后，不按体例分先后。
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
        <input v-model="keyword" type="search" placeholder="搜标题、正文、地点" aria-label="搜索实录" />
        <button type="submit" class="text-link">检</button>
      </form>
    </div>

    <div class="result-line">
      <span class="muted">共 {{ total }} 条</span>
      <span class="rule"></span>
    </div>

    <div v-if="list.length" class="entry-list">
      <RouterLink v-for="e in list" :key="e.id" class="entry" :to="`/events/${e.id}`">
        <div class="entry-date">
          <b>{{ e.year }}</b>
          <span>{{ e.month ? `${e.month} 月` : '年份未详' }}{{ e.day ? ` ${e.day} 日` : '' }}</span>
        </div>
        <div class="entry-main">
          <h3>{{ e.title }}</h3>
          <p>{{ e.excerpt }}</p>
          <div class="entry-tags">
            <span class="branch-link">{{ e.branchName }}</span>
            <span v-if="e.inherited" class="chip">承继</span>
            <span v-for="t in e.tags.slice(0, 3)" :key="t" class="chip">#{{ t }}</span>
            <span v-if="e.location" class="chip chip-dai">@{{ e.location }}</span>
          </div>
        </div>
        <div class="entry-side">
          <span>{{ shortDate(e.year, e.month, e.day) }}</span>
          <span>{{ e.likeCount }} 赞</span>
        </div>
      </RouterLink>
    </div>
    <div v-else-if="!loading" class="empty">未检得相关实录。</div>

    <div v-if="totalPages > 1" class="pager">
      <button class="btn btn-sm btn-ghost" :disabled="page <= 1" @click="goPage(page - 1)">上一页</button>
      <span class="pager-num">第 {{ page }} / {{ totalPages }} 页</span>
      <button class="btn btn-sm btn-ghost" :disabled="page >= totalPages" @click="goPage(page + 1)">下一页</button>
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

.result-line {
  display: flex;
  align-items: center;
  gap: 14px;
  margin: 22px 0 0;
}

.result-line .rule {
  flex: 1;
  height: 1px;
  background: var(--rule);
}

.pager {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 18px;
  margin: 40px 0 20px;
}

.pager-num {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--ink-3);
}

.entry-side {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

@media (max-width: 720px) {
  .entry-side {
    display: none;
  }
}
</style>

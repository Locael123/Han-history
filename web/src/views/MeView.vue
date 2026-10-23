<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api'
import type { BranchCard, TimelineItem } from '@/api/types'
import BranchCardView from '@/components/BranchCard.vue'
import { useAuthStore } from '@/stores/auth'
import { shortDate, timeAgo } from '@/utils/format'

const auth = useAuthStore()

const branches = ref<BranchCard[]>([])
const events = ref<TimelineItem[]>([])
const tab = ref<'branches' | 'events'>('branches')
const loading = ref(true)

const me = computed(() => auth.me)
const joined = computed(() => {
  const raw = me.value?.createdAt
  if (!raw) return ''
  const d = new Date(raw)
  return `${d.getFullYear()} 年 ${d.getMonth() + 1} 月`
})

const totalEvents = computed(() => branches.value.reduce((sum, b) => sum + b.ownCount, 0))
const totalLikes = computed(() => branches.value.reduce((sum, b) => sum + b.likeCount, 0))

onMounted(async () => {
  try {
    await auth.fetchMe()
    const [b, e] = await Promise.all([api.myBranches(1, 50), api.myEvents(1, 50)])
    branches.value = b.list
    events.value = e.list
  } catch (error) {
    ElMessage.error((error as Error).message)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="page-wrap me-page" v-loading="loading">
    <header class="me-head">
      <div class="me-seal">{{ (auth.displayName || '史').slice(0, 1) }}</div>
      <div class="me-main">
        <div class="kicker">我的史稿</div>
        <h2>{{ auth.displayName }}</h2>
        <div class="me-meta">
          <span>@{{ me?.username }}</span>
          <span class="sep">·</span>
          <span>{{ joined }}入史</span>
          <span v-if="me?.bio" class="sep">·</span>
          <span v-if="me?.bio">{{ me.bio }}</span>
        </div>
      </div>
      <RouterLink class="btn btn-sm btn-ghost" to="/branches">去分叉</RouterLink>
    </header>

    <div class="me-stats">
      <div>
        <b>{{ branches.length }}</b>
        <span>所立之枝</span>
      </div>
      <div>
        <b>{{ totalEvents }}</b>
        <span>自写条目</span>
      </div>
      <div>
        <b>{{ totalLikes }}</b>
        <span>枝上获赞</span>
      </div>
      <div>
        <b>{{ events.length }}</b>
        <span>近录条目</span>
      </div>
    </div>

    <div class="tabs">
      <button :class="{ on: tab === 'branches' }" @click="tab = 'branches'">
        我的史枝 <span class="serif-num">{{ branches.length }}</span>
      </button>
      <button :class="{ on: tab === 'events' }" @click="tab = 'events'">
        我的条目 <span class="serif-num">{{ events.length }}</span>
      </button>
    </div>

    <template v-if="tab === 'branches'">
      <div v-if="branches.length" class="branch-grid">
        <BranchCardView v-for="b in branches" :key="b.id" :branch="b" />
      </div>
      <div v-else-if="!loading" class="empty">
        你还没有立过枝。去任意一枝，在你想接续的那条实录旁点「自此分叉」。
        <div style="margin-top: 18px">
          <RouterLink class="btn btn-sm btn-cinnabar" to="/graph">去谱系</RouterLink>
        </div>
      </div>
    </template>

    <template v-else>
      <div v-if="events.length" class="entry-list">
        <RouterLink v-for="e in events" :key="e.id" class="entry" :to="`/events/${e.id}`">
          <div class="entry-date">
            <b>{{ e.year }}</b>
            <span>{{ e.month ? `${e.month} 月` : '年份未详' }}</span>
          </div>
          <div class="entry-main">
            <h3>{{ e.title }}</h3>
            <p>{{ e.excerpt }}</p>
            <div class="entry-tags">
              <span class="branch-link">{{ e.branchName }}</span>
              <span class="chip">{{ e.likeCount }} 赞</span>
              <span class="chip">{{ timeAgo(e.createdAt) }}</span>
            </div>
          </div>
          <div class="entry-side">
            <span>{{ shortDate(e.year, e.month, e.day) }}</span>
          </div>
        </RouterLink>
      </div>
      <div v-else-if="!loading" class="empty">还没有写过条目。</div>
    </template>
  </div>
</template>

<style scoped>
.me-head {
  display: flex;
  align-items: center;
  gap: 22px;
  border-bottom: 1px solid var(--ink);
  padding: 20px 0 26px;
}

.me-seal {
  width: 64px;
  height: 64px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--cinnabar);
  color: #f7f1e3;
  font-family: var(--serif);
  font-size: 30px;
  border-radius: 2px;
}

.me-main {
  flex: 1;
  min-width: 0;
}

.me-main h2 {
  font-family: var(--serif);
  font-size: 34px;
  font-weight: 500;
  letter-spacing: 0.1em;
  margin: 8px 0 8px;
}

.me-meta {
  font-size: 12px;
  color: var(--ink-3);
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.me-meta .sep {
  color: var(--ink-4);
}

.me-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  border-bottom: 1px solid var(--rule);
}

.me-stats > div {
  padding: 22px 0;
  border-right: 1px solid var(--rule-2);
  padding-left: 4px;
}

.me-stats > div:last-child {
  border-right: 0;
}

.me-stats b {
  display: block;
  font-family: var(--serif);
  font-size: 32px;
  font-weight: 500;
  line-height: 1;
}

.me-stats span {
  display: block;
  margin-top: 8px;
  font-size: 11px;
  letter-spacing: 0.12em;
  color: var(--ink-3);
}

.tabs {
  margin: 30px 0 24px;
}

@media (max-width: 720px) {
  .me-stats {
    grid-template-columns: 1fr 1fr;
  }

  .me-stats > div:nth-child(2) {
    border-right: 0;
  }
}
</style>

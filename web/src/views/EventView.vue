<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '@/api'
import type { EventDetail, TimelineItem } from '@/api/types'
import CommentSection from '@/components/CommentSection.vue'
import EventDialog from '@/components/EventDialog.vue'
import ForkDialog from '@/components/ForkDialog.vue'
import { useAuthStore } from '@/stores/auth'
import { branchTone, longDate, republicYear, shortDate } from '@/utils/format'

const props = defineProps<{ id: string }>()
const router = useRouter()
const auth = useAuthStore()

const eventId = computed(() => Number(props.id))
const detail = ref<EventDetail | null>(null)
const loading = ref(true)
const likeBusy = ref(false)
const neighbors = ref<TimelineItem[]>([])
const eventDialog = ref(false)
const forkDialog = ref(false)

const paragraphs = computed(() =>
  (detail.value?.content || '')
    .split(/\n+/)
    .map((p) => p.trim())
    .filter(Boolean),
)

const tone = computed(() => (detail.value ? branchTone(detail.value.branchOfficial, detail.value.branchId) : 'var(--ink)'))

const currentIndex = computed(() => neighbors.value.findIndex((n) => n.id === eventId.value))
const prev = computed(() => (currentIndex.value > 0 ? neighbors.value[currentIndex.value - 1] : null))
const next = computed(() =>
  currentIndex.value >= 0 && currentIndex.value < neighbors.value.length - 1
    ? neighbors.value[currentIndex.value + 1]
    : null,
)

async function load() {
  loading.value = true
  try {
    detail.value = await api.event(eventId.value)
    const tl = await api.branchTimeline(detail.value.branchId, 1, 100)
    neighbors.value = tl.list
  } catch (e) {
    ElMessage.error((e as Error).message)
    router.replace('/events')
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(eventId, load)

async function toggleLike() {
  if (!detail.value) return
  if (!auth.isLoggedIn) {
    ElMessage.warning('登录后才能点赞')
    return
  }
  likeBusy.value = true
  try {
    const res = await api.toggleLike('event', detail.value.id)
    detail.value.liked = res.liked
    detail.value.likeCount = res.count
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    likeBusy.value = false
  }
}

function setCommentCount(count: number) {
  if (detail.value) detail.value.commentCount = count
}

function openFork() {
  if (!auth.isLoggedIn) {
    ElMessage.warning('登录后才能分叉')
    router.push({ name: 'login', query: { redirect: `/events/${eventId.value}` } })
    return
  }
  forkDialog.value = true
}

function onForked(id: number) {
  ElMessage.success('新枝已立')
  router.push(`/branches/${id}`)
}

function onSaved(saved: EventDetail) {
  detail.value = saved
  eventDialog.value = false
}

async function removeEvent() {
  if (!detail.value) return
  const branchId = detail.value.branchId
  try {
    await ElMessageBox.confirm('删去之后不可复得，确定？', '删此条', {
      confirmButtonText: '删',
      cancelButtonText: '留着',
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await api.deleteEvent(eventId.value)
    ElMessage.success('已删')
    router.push(`/branches/${branchId}`)
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}
</script>

<template>
  <div v-loading="loading" class="page-wrap event-page">
    <template v-if="detail">
      <nav class="crumbs">
        <RouterLink to="/events">实录</RouterLink>
        <span>/</span>
        <RouterLink :to="`/branches/${detail.branchId}`">{{ detail.branchName }}</RouterLink>
        <span>/</span>
        <span>{{ detail.title }}</span>
      </nav>

      <article class="article">
        <header class="article-head">
          <div class="year-block">
            <b>{{ detail.year }}</b>
            <span>{{ republicYear(detail.year) }}</span>
          </div>
          <div class="head-main">
            <div class="kicker">
              <RouterLink class="branch-chip" :to="`/branches/${detail.branchId}`" :style="{ '--branch-color': tone }">
                {{ detail.branchName }}
              </RouterLink>
              <span v-if="detail.inherited" class="muted">承继条目</span>
              <span v-else class="muted">本枝自记</span>
            </div>
            <h1 class="article-title">{{ detail.title }}</h1>
            <div class="article-meta">
              <span>{{ longDate(detail.year, detail.month, detail.day) }}</span>
              <span v-if="detail.location">· {{ detail.location }}</span>
              <span class="sep">·</span>
              <span>录者 {{ detail.author?.nickname || '史馆' }}</span>
            </div>
          </div>
        </header>

        <div class="article-rule"></div>

        <div class="article-body">
          <p v-for="(p, i) in paragraphs" :key="i">{{ p }}</p>
          <p v-if="!paragraphs.length" class="muted">（此条无正文）</p>
        </div>

        <div v-if="detail.figures.length || detail.tags.length" class="article-facts">
          <div v-if="detail.figures.length" class="fact">
            <span class="fact-label">人物</span>
            <span>{{ detail.figures.join('　') }}</span>
          </div>
          <div v-if="detail.tags.length" class="fact">
            <span class="fact-label">标签</span>
            <span class="chips">
              <span v-for="t in detail.tags" :key="t" class="chip">#{{ t }}</span>
            </span>
          </div>
        </div>

        <div class="article-actions">
          <button class="btn btn-sm" :class="{ 'is-on': detail.liked }" :disabled="likeBusy" @click="toggleLike">
            {{ detail.liked ? '已赞' : '赞' }} {{ detail.likeCount }}
          </button>
          <a class="btn btn-sm btn-ghost" href="#comments">评论 {{ detail.commentCount }}</a>
          <button class="btn btn-sm btn-ghost" @click="openFork">自此分叉</button>
          <span class="spacer"></span>
          <template v-if="detail.canEdit">
            <button class="btn btn-sm btn-ghost" @click="eventDialog = true">改</button>
            <button class="btn btn-sm btn-ghost danger" @click="removeEvent">删</button>
          </template>
        </div>

        <div class="event-pager">
          <RouterLink v-if="prev" :to="`/events/${prev.id}`" class="pager-side">
            <span class="muted">← 前一条</span>
            <b>{{ prev.title }}</b>
            <em>{{ shortDate(prev.year, prev.month, prev.day) }}</em>
          </RouterLink>
          <span v-else></span>
          <RouterLink v-if="next" :to="`/events/${next.id}`" class="pager-side right">
            <span class="muted">后一条 →</span>
            <b>{{ next.title }}</b>
            <em>{{ shortDate(next.year, next.month, next.day) }}</em>
          </RouterLink>
        </div>
      </article>

      <div id="comments" class="comments-wrap">
        <CommentSection target-type="event" :target-id="detail.id" @change="setCommentCount" />
      </div>

      <ForkDialog
        v-model="forkDialog"
        :parent-id="detail.branchId"
        :parent-name="detail.branchName"
        :fork-event-id="detail.id"
        :fork-event-title="detail.title"
        :fork-year="detail.year"
        @forked="(b) => onForked(b.id)"
      />

      <EventDialog
        v-model="eventDialog"
        :branch-id="detail.branchId"
        :branch-name="detail.branchName"
        :event="detail"
        @saved="onSaved"
      />
    </template>
  </div>
</template>

<style scoped>
.crumbs {
  display: flex;
  gap: 8px;
  font-size: 11.5px;
  color: var(--ink-3);
  letter-spacing: 0.06em;
  padding: 16px 0 26px;
  max-width: 760px;
  margin: 0 auto;
}

.crumbs a:hover {
  color: var(--cinnabar);
}

.article-head {
  display: flex;
  gap: 26px;
  align-items: flex-start;
  text-align: left;
  border-top: 2px solid var(--ink);
  padding-top: 14px;
}

.year-block {
  flex: none;
  width: 92px;
  border-top: 0;
  padding-top: 0;
}

.year-block b {
  display: block;
  font-family: var(--serif);
  font-size: 42px;
  line-height: 1;
  letter-spacing: 0.02em;
}

.year-block span {
  display: block;
  margin-top: 8px;
  font-size: 10.5px;
  color: var(--ink-3);
  letter-spacing: 0.1em;
}

.head-main {
  flex: 1;
  min-width: 0;
  border-top: 0;
  padding-top: 0;
}

.article-title {
  font-family: var(--serif);
  font-size: clamp(28px, 3.8vw, 42px);
  font-weight: 500;
  line-height: 1.36;
  letter-spacing: 0.04em;
  margin: 12px 0 14px;
}

.article-meta {
  font-size: 12px;
  color: var(--ink-3);
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.article-meta .sep {
  color: var(--ink-4);
}

.article-rule {
  height: 1px;
  background: var(--rule);
  margin: 26px 0 30px;
}

.article-body p {
  font-family: var(--serif);
  font-size: 16.5px;
  line-height: 2.15;
  color: var(--ink);
  margin: 0 0 1.1em;
  letter-spacing: 0.01em;
  text-align: justify;
}

.article-facts {
  margin: 32px 0 0;
  border-top: 1px solid var(--rule-2);
  border-bottom: 1px solid var(--rule-2);
  padding: 14px 0;
}

.fact {
  display: flex;
  gap: 18px;
  font-size: 13px;
  color: var(--ink-2);
  padding: 4px 0;
}

.fact-label {
  flex: none;
  width: 44px;
  color: var(--ink-3);
  font-size: 11px;
  letter-spacing: 0.16em;
  padding-top: 2px;
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.article-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 26px;
}

.article-actions .spacer {
  flex: 1;
}

.article-actions .danger:hover {
  color: var(--cinnabar);
  border-color: var(--cinnabar);
}

.event-pager {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  margin-top: 56px;
  border-top: 1px solid var(--rule);
  padding-top: 18px;
}

.pager-side {
  display: flex;
  flex-direction: column;
  gap: 5px;
  font-size: 13px;
  color: var(--ink-2);
}

.pager-side b {
  font-family: var(--serif);
  font-weight: 500;
  font-size: 15px;
}

.pager-side em {
  font-style: normal;
  font-family: var(--mono);
  font-size: 10.5px;
  color: var(--ink-3);
}

.pager-side.right {
  text-align: right;
}

.comments-wrap {
  max-width: 760px;
  margin: 0 auto;
  padding-bottom: 40px;
}

.branch-chip {
  display: inline-block;
  border-left: 3px solid var(--branch-color);
  padding-left: 8px;
  color: var(--ink);
  font-family: var(--serif);
  font-size: 13px;
  letter-spacing: 0.06em;
}

.branch-chip:hover {
  color: var(--cinnabar);
}

@media (max-width: 720px) {
  .article-head {
    flex-direction: column;
    gap: 14px;
  }

  .year-block {
    width: auto;
    display: flex;
    align-items: baseline;
    gap: 12px;
  }

  .year-block b {
    font-size: 30px;
  }

  .event-pager {
    grid-template-columns: 1fr;
  }

  .pager-side.right {
    text-align: left;
  }
}
</style>

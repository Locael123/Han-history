<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '@/api'
import type { BranchDetail, EventDetail, TimelineItem } from '@/api/types'
import CommentSection from '@/components/CommentSection.vue'
import EventDialog from '@/components/EventDialog.vue'
import ForkDialog from '@/components/ForkDialog.vue'
import { useAuthStore } from '@/stores/auth'
import { branchTone, republicYear, shortDate } from '@/utils/format'

const props = defineProps<{ id: string }>()
const router = useRouter()
const auth = useAuthStore()

const branchId = computed(() => Number(props.id))
const detail = ref<BranchDetail | null>(null)
const timeline = ref<TimelineItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(true)
const loadingMore = ref(false)
const likeBusy = ref(false)
const ownOnly = ref(true)
const ORDER_KEY = 'gonghe-timeline-desc'
const desc = ref(localStorage.getItem(ORDER_KEY) === '1')

const eventDialog = ref(false)
const editingEvent = ref<EventDetail | null>(null)
const forkDialog = ref(false)
const forkTarget = ref({ eventId: 0, title: '', year: 0 })

const editDialog = ref(false)
const editName = ref('')
const editDesc = ref('')
const savingBranch = ref(false)

const tone = computed(() => (detail.value ? branchTone(detail.value.official, detail.value.id) : 'var(--ink)'))
const ownerName = computed(() => detail.value?.owner?.nickname || '史馆')
const canWrite = computed(() => !!detail.value && !detail.value.official && detail.value.isOwner)
const canManage = computed(() => !!detail.value && detail.value.isOwner)

async function load() {
  loading.value = true
  page.value = 1
  try {
    const [d, t] = await Promise.all([
      api.branch(branchId.value),
      api.branchTimeline(branchId.value, 1, pageSize, ownOnly.value, desc.value),
    ])
    detail.value = d
    timeline.value = t.list
    total.value = t.total
  } catch (e) {
    ElMessage.error((e as Error).message)
    router.replace('/branches')
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(branchId, load)

function switchOwn(value: boolean) {
  if (ownOnly.value === value) return
  ownOnly.value = value
  load()
}

function switchOrder(value: boolean) {
  if (desc.value === value) return
  desc.value = value
  localStorage.setItem(ORDER_KEY, value ? '1' : '0')
  load()
}

function eventOrder(a: TimelineItem, b: TimelineItem): number {
  return a.year * 10000 + a.month * 100 + a.day - (b.year * 10000 + b.month * 100 + b.day) || a.id - b.id
}

function resortTimeline() {
  timeline.value.sort((a, b) => (desc.value ? -eventOrder(a, b) : eventOrder(a, b)))
}

async function loadMore() {
  loadingMore.value = true
  try {
    const next = page.value + 1
    const t = await api.branchTimeline(branchId.value, next, pageSize, ownOnly.value, desc.value)
    timeline.value = [...timeline.value, ...t.list]
    total.value = t.total
    page.value = next
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loadingMore.value = false
  }
}

async function toggleLike() {
  if (!detail.value) return
  if (!auth.isLoggedIn) {
    ElMessage.warning('登录后才能点赞')
    return
  }
  likeBusy.value = true
  try {
    const res = await api.toggleLike('branch', detail.value.id)
    detail.value.liked = res.liked
    detail.value.likeCount = res.count
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    likeBusy.value = false
  }
}

function openFork(eventId: number, title: string, year: number) {
  if (!auth.isLoggedIn) {
    ElMessage.warning('登录后才能分叉')
    router.push({ name: 'login', query: { redirect: `/branches/${branchId.value}` } })
    return
  }
  forkTarget.value = { eventId, title, year }
  forkDialog.value = true
}

function onForked(id: number) {
  ElMessage.success('新枝已立')
  router.push(`/branches/${id}`)
}

function openCreate() {
  if (!canWrite.value) {
    if (detail.value?.official) {
      ElMessage.info('官修正史不可续写，请先分叉')
    } else {
      ElMessage.info('只有史枝主人可以续写')
    }
    return
  }
  editingEvent.value = null
  eventDialog.value = true
}

async function openEdit(item: TimelineItem) {
  try {
    editingEvent.value = await api.event(item.id)
    eventDialog.value = true
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function onSaved(saved: EventDetail) {
  if (editingEvent.value) {
    const idx = timeline.value.findIndex((t) => t.id === saved.id)
    if (idx >= 0) timeline.value[idx] = saved
    const d = detail.value
    if (d) {
      d.firstYear = d.firstYear ? Math.min(d.firstYear, saved.year) : saved.year
      d.lastYear = Math.max(d.lastYear, saved.year)
    }
  } else {
    timeline.value = [saved, ...timeline.value]
    total.value += 1
    if (detail.value) {
      detail.value.eventCount += 1
      detail.value.ownCount += 1
    }
  }
  resortTimeline()
  editingEvent.value = null
}

async function removeEvent(item: TimelineItem) {
  try {
    await ElMessageBox.confirm(`删去《${item.title}》之后不可复得，确定？`, '删此条', {
      confirmButtonText: '删',
      cancelButtonText: '留着',
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await api.deleteEvent(item.id)
    timeline.value = timeline.value.filter((t) => t.id !== item.id)
    total.value = Math.max(0, total.value - 1)
    if (detail.value) {
      detail.value.eventCount = Math.max(0, detail.value.eventCount - 1)
      detail.value.ownCount = Math.max(0, detail.value.ownCount - 1)
    }
    ElMessage.success('已删')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function openEditBranch() {
  if (!detail.value) return
  editName.value = detail.value.name
  editDesc.value = detail.value.description
  editDialog.value = true
}

async function saveBranch() {
  if (!detail.value) return
  const n = editName.value.trim()
  if (n.length < 2 || n.length > 40) {
    ElMessage.warning('枝名需 2 至 40 字')
    return
  }
  savingBranch.value = true
  try {
    const updated = await api.updateBranch(detail.value.id, { name: n, description: editDesc.value.trim() })
    detail.value = updated
    editDialog.value = false
    ElMessage.success('已改定')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    savingBranch.value = false
  }
}

function setCommentCount(count: number) {
  if (detail.value) detail.value.commentCount = count
}

async function removeBranch() {
  if (!detail.value) return
  try {
    await ElMessageBox.confirm('删去整枝，连同其下所有条目与评论。确定？', '删此枝', {
      confirmButtonText: '删',
      cancelButtonText: '留着',
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await api.deleteBranch(detail.value.id)
    ElMessage.success('枝已折去')
    router.push('/branches')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}
</script>

<template>
  <div v-loading="loading" class="page-wrap branch-page">
    <template v-if="detail">
      <nav class="crumbs">
        <RouterLink to="/branches">史枝总目</RouterLink>
        <span>/</span>
        <span>{{ detail.name }}</span>
      </nav>

      <header class="branch-hero" :style="{ '--branch-color': tone }">
        <div class="hero-mark"></div>
        <div class="branch-hero-main">
          <div class="kicker">
            {{ detail.official ? '官修正史' : '民间野史' }} · {{ detail.parentId ? '分叉枝' : '主干' }}
          </div>
          <h2 class="branch-title">{{ detail.name }}</h2>
          <p class="branch-desc">{{ detail.description || '此枝尚无小引。' }}</p>
          <div class="branch-meta">
            <span>主笔 {{ ownerName }}</span>
            <span class="sep">·</span>
            <span>{{ detail.eventCount }} 条实录（承继 {{ detail.inheritedCount }} / 自有 {{ detail.ownCount }}）</span>
            <span class="sep">·</span>
            <span>{{ detail.forkCount }} 处分叉</span>
            <span class="sep">·</span>
            <span>{{ detail.firstYear || '—' }}–{{ detail.lastYear || '—' }}</span>
          </div>
          <div class="branch-actions">
            <button class="btn btn-sm" :class="{ 'is-on': detail.liked }" :disabled="likeBusy" @click="toggleLike">
              {{ detail.liked ? '已赞' : '赞' }} {{ detail.likeCount }}
            </button>
            <button v-if="canWrite" class="btn btn-sm btn-primary" @click="openCreate">续写一条</button>
            <button v-else-if="detail.official" class="btn btn-sm btn-ghost" @click="openFork(0, '', 0)">
              自分叉处续写
            </button>
            <button v-else class="btn btn-sm btn-ghost" @click="openFork(0, '', 0)">自此分叉</button>
            <template v-if="canManage">
              <button class="btn btn-sm btn-ghost" @click="openEditBranch">改枝名</button>
              <button class="btn btn-sm btn-ghost danger" @click="removeBranch">删枝</button>
            </template>
          </div>
        </div>
      </header>

      <div class="detail-layout">
        <main>
          <section class="section">
            <div class="section-head">
              <span class="ordinal">纪</span>
              <h2>本枝时间轴</h2>
              <span class="rule"></span>
              <span v-if="!detail.parentId" class="hint">{{ detail.firstYear || '—' }} — {{ detail.lastYear || '—' }}</span>
              <div class="tl-controls">
                <div v-if="detail.parentId" class="seg">
                  <button :class="{ on: !ownOnly }" @click="switchOwn(false)">含承继</button>
                  <button :class="{ on: ownOnly }" @click="switchOwn(true)">本枝自记</button>
                </div>
                <div class="seg">
                  <button :class="{ on: !desc }" @click="switchOrder(false)">正序</button>
                  <button :class="{ on: desc }" @click="switchOrder(true)">倒叙</button>
                </div>
              </div>
            </div>

            <div v-if="timeline.length" class="timeline">
              <article
                v-for="item in timeline"
                :key="item.id"
                class="tl-item"
                :class="{ own: !item.inherited, 'is-inherited': item.inherited }"
              >
                <div class="tl-year">
                  <b>{{ item.year }}</b>
                  <span>{{ item.month ? `${item.month} 月` : '—' }}{{ item.day ? ` ${item.day} 日` : '' }}</span>
                  <em>{{ republicYear(item.year) }}</em>
                </div>
                <div class="tl-body">
                  <div class="tl-tagline">
                    <RouterLink v-if="item.inherited" class="chip" :to="`/branches/${item.branchId}`">
                      承继 · {{ item.branchName }}
                    </RouterLink>
                    <span v-else class="chip chip-red">本枝自记</span>
                    <span v-for="t in item.tags.slice(0, 3)" :key="t" class="chip chip-ink">#{{ t }}</span>
                    <span v-if="item.location" class="muted location">@{{ item.location }}</span>
                  </div>
                  <RouterLink :to="`/events/${item.id}`" class="tl-title">{{ item.title }}</RouterLink>
                  <p class="tl-text">{{ item.excerpt }}</p>
                  <div class="tl-foot">
                    <span class="muted">{{ item.author?.nickname || '史馆' }}</span>
                    <span class="muted">{{ shortDate(item.year, item.month, item.day) }}</span>
                    <span class="muted">{{ item.likeCount }} 赞 · {{ item.commentCount }} 评</span>
                    <span class="spacer"></span>
                    <button type="button" @click="openFork(item.id, item.title, item.year)">自此分叉</button>
                    <template v-if="canWrite && !item.inherited">
                      <button type="button" @click="openEdit(item)">改</button>
                      <button type="button" class="danger" @click="removeEvent(item)">删</button>
                    </template>
                    <RouterLink class="text-link" :to="`/events/${item.id}`">读全文 →</RouterLink>
                  </div>
                </div>
              </article>
            </div>
            <div v-else-if="!loading" class="empty">
              {{ ownOnly ? '此枝尚无自记的条目，点「续写一条」另起一笔。' : '此枝尚无实录。' }}
            </div>

            <div v-if="timeline.length < total" class="load-more">
              <button class="btn btn-sm btn-ghost" :disabled="loadingMore" @click="loadMore">
                {{ loadingMore ? '载入中…' : `再展二十条（余 ${total - timeline.length}）` }}
              </button>
            </div>
          </section>

          <CommentSection
            target-type="branch"
            :target-id="detail.id"
            @change="setCommentCount"
          />
        </main>

        <aside class="branch-aside">
          <div class="aside-block">
            <div class="kicker">谱系</div>
            <div class="lineage-path">
              <template v-for="(a, i) in detail.ancestors" :key="a.id">
                <RouterLink :to="`/branches/${a.id}`">{{ a.name }}</RouterLink>
                <span class="arrow">→</span>
                <template v-if="i === detail.ancestors.length - 1">
                  <span class="now">{{ detail.name }}</span>
                </template>
              </template>
              <span v-if="!detail.ancestors.length" class="now">{{ detail.name }}（主干）</span>
            </div>
            <p v-if="detail.forkEventTitle" class="aside-note">
              分叉于 {{ detail.forkYear }} 年《{{ detail.forkEventTitle }}》之后，承继此前 {{ detail.inheritedCount }} 条。
            </p>
          </div>

          <div v-if="detail.children.length" class="aside-block">
            <div class="kicker">自此分出的枝</div>
            <RouterLink v-for="c in detail.children" :key="c.id" :to="`/branches/${c.id}`" class="child-line">
              <span class="dot" :style="{ background: branchTone(c.official, c.id) }"></span>
              <span class="child-name">{{ c.name }}</span>
              <em>{{ c.forkYear }}</em>
            </RouterLink>
          </div>

          <div v-if="detail.siblings.length" class="aside-block">
            <div class="kicker">同源别枝</div>
            <RouterLink v-for="c in detail.siblings" :key="c.id" :to="`/branches/${c.id}`" class="child-line">
              <span class="dot" :style="{ background: branchTone(c.official, c.id) }"></span>
              <span class="child-name">{{ c.name }}</span>
              <em>{{ c.forkYear }}</em>
            </RouterLink>
          </div>

          <div class="aside-block">
            <div class="kicker">枝数</div>
            <dl class="aside-stats">
              <div><dt>实录</dt><dd>{{ detail.eventCount }}</dd></div>
              <div><dt>承继</dt><dd>{{ detail.inheritedCount }}</dd></div>
              <div><dt>自有</dt><dd>{{ detail.ownCount }}</dd></div>
              <div><dt>分叉</dt><dd>{{ detail.forkCount }}</dd></div>
              <div><dt>赞</dt><dd>{{ detail.likeCount }}</dd></div>
              <div><dt>评论</dt><dd>{{ detail.commentCount }}</dd></div>
            </dl>
          </div>

          <div class="aside-block">
            <div class="kicker">立枝</div>
            <p class="aside-note">分支之后，此前的条目照旧承继；自此以后，各写各的。</p>
            <button class="btn btn-sm btn-cinnabar" @click="openFork(0, '', 0)">自此分叉</button>
          </div>
        </aside>
      </div>

      <ForkDialog
        v-model="forkDialog"
        :parent-id="detail.id"
        :parent-name="detail.name"
        :fork-event-id="forkTarget.eventId"
        :fork-event-title="forkTarget.title"
        :fork-year="forkTarget.year || detail.lastYear"
        @forked="(b) => onForked(b.id)"
      />

      <EventDialog
        v-model="eventDialog"
        :branch-id="detail.id"
        :branch-name="detail.name"
        :event="editingEvent"
        @saved="onSaved"
      />

      <el-dialog v-model="editDialog" width="480px" align-center class="gonghe-dialog" title="改定枝名">
        <div class="field">
          <label>枝名</label>
          <input v-model="editName" maxlength="40" />
        </div>
        <div class="field">
          <label>小引</label>
          <textarea v-model="editDesc" rows="4" maxlength="300"></textarea>
        </div>
        <template #footer>
          <button class="btn btn-sm btn-ghost" @click="editDialog = false">且慢</button>
          <button class="btn btn-sm btn-primary" :disabled="savingBranch" @click="saveBranch">改定</button>
        </template>
      </el-dialog>
    </template>
  </div>
</template>

<style scoped>
.tl-controls {
  display: flex;
  align-items: center;
  gap: 10px;
}

@media (max-width: 720px) {
  .tl-controls {
    width: 100%;
    justify-content: flex-end;
  }
}

.crumbs {
  display: flex;
  gap: 8px;
  font-size: 11.5px;
  color: var(--ink-3);
  letter-spacing: 0.08em;
  padding: 16px 0 20px;
}

.crumbs a:hover {
  color: var(--cinnabar);
}

.branch-hero {
  position: relative;
  display: flex;
  gap: 24px;
  padding: 8px 0 34px;
  border-bottom: 1px solid var(--ink);
}

.hero-mark {
  width: 4px;
  align-self: stretch;
  background: var(--branch-color);
  flex: none;
}

.branch-hero-main {
  flex: 1;
  min-width: 0;
}

.branch-title {
  font-family: var(--serif);
  font-size: clamp(34px, 5vw, 58px);
  font-weight: 500;
  letter-spacing: 0.08em;
  line-height: 1.25;
  margin: 10px 0 14px;
}

.branch-desc {
  max-width: 720px;
  font-size: 15px;
  line-height: 2;
  color: var(--ink-2);
  margin: 0 0 18px;
}

.branch-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 12px;
  color: var(--ink-3);
}

.branch-meta .sep {
  color: var(--ink-4);
}

.branch-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 20px;
}

.branch-actions .danger:hover {
  color: var(--cinnabar);
  border-color: var(--cinnabar);
}

.branch-aside {
  position: sticky;
  top: 24px;
  align-self: start;
}

.lineage-path {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px;
  font-family: var(--serif);
  font-size: 14px;
  line-height: 1.9;
  color: var(--ink-2);
}

.lineage-path .arrow {
  color: var(--ink-4);
}

.lineage-path .now {
  color: var(--cinnabar);
}

.child-line {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 9px 0;
  border-bottom: 1px solid var(--rule-2);
  font-size: 13px;
  color: var(--ink-2);
}

.child-line .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex: none;
}

.child-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--serif);
}

.child-line em {
  font-style: normal;
  font-family: var(--mono);
  font-size: 10px;
  color: var(--ink-3);
}

.location {
  margin-left: auto;
}

.tl-title {
  display: block;
  font-family: var(--serif);
  font-size: 20px;
  letter-spacing: 0.03em;
  color: var(--ink);
  margin: 8px 0 6px;
}

.tl-title:hover {
  color: var(--cinnabar);
}

.tl-foot .danger:hover {
  color: var(--cinnabar);
}

.tl-foot .spacer {
  flex: 1;
}

.load-more {
  text-align: center;
  margin-top: 24px;
}

.detail-layout {
  align-items: start;
}

@media (max-width: 980px) {
  .branch-aside {
    position: static;
    margin-top: 40px;
  }
}
</style>

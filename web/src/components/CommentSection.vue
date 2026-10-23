<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '@/api'
import type { CommentView } from '@/api/types'
import { useAuthStore } from '@/stores/auth'
import { initialOf, timeAgo } from '@/utils/format'

const props = defineProps<{
  targetType: 'branch' | 'event'
  targetId: number
}>()

const emit = defineEmits<{ (e: 'change', count: number): void }>()

const auth = useAuthStore()
const route = useRoute()

const list = ref<CommentView[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)
const submitting = ref(false)
const content = ref('')
const replyTo = ref<{ id: number; name: string } | null>(null)
const textarea = ref<HTMLTextAreaElement | null>(null)

async function load(reset = false) {
  if (reset) page.value = 1
  loading.value = true
  try {
    const res = await api.comments(props.targetType, props.targetId, page.value, pageSize)
    list.value = reset ? res.list : [...list.value, ...res.list]
    total.value = res.total
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

onMounted(() => load(true))
watch(() => [props.targetType, props.targetId], () => load(true))

function startReply(c: CommentView) {
  replyTo.value = { id: c.id, name: c.user?.nickname || c.user?.username || '无名' }
  nextTick(() => textarea.value?.focus())
}

function cancelReply() {
  replyTo.value = null
}

async function submit() {
  const text = content.value.trim()
  if (!text) {
    ElMessage.warning('写点什么再落笔')
    return
  }
  if (text.length > 500) {
    ElMessage.warning('评论不宜超过 500 字')
    return
  }
  submitting.value = true
  try {
    const c = await api.createComment({
      targetType: props.targetType,
      targetId: props.targetId,
      content: text,
      replyToId: replyTo.value?.id || 0,
    })
    list.value = [c, ...list.value]
    total.value += 1
    emit('change', total.value)
    content.value = ''
    replyTo.value = null
    ElMessage.success('已记下')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    submitting.value = false
  }
}

async function like(c: CommentView) {
  if (!auth.isLoggedIn) {
    ElMessage.warning('登录后才能点赞')
    return
  }
  try {
    const res = await api.toggleLike('comment', c.id)
    c.liked = res.liked
    c.likeCount = res.count
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function remove(c: CommentView) {
  try {
    await ElMessageBox.confirm('删去之后不可复得，确定？', '删此评', {
      confirmButtonText: '删',
      cancelButtonText: '留着',
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await api.deleteComment(c.id)
    list.value = list.value.filter((item) => item.id !== c.id)
    total.value = Math.max(0, total.value - 1)
    emit('change', total.value)
    ElMessage.success('已删')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function loadMore() {
  page.value += 1
  load()
}

const loginLink = () => ({ name: 'login', query: { redirect: route.fullPath } })
</script>

<template>
  <section class="comments">
    <div class="comments-head">
      <h3>评论 <span class="serif-num">{{ total }}</span></h3>
      <span class="hint">只论此事，不必论人</span>
    </div>

    <form v-if="auth.isLoggedIn" class="comment-form" @submit.prevent="submit">
      <textarea
        ref="textarea"
        v-model="content"
        rows="4"
        maxlength="500"
        :placeholder="replyTo ? `回复 ${replyTo.name}：` : '写下你的按语……'"
      ></textarea>
      <div class="comment-form-foot">
        <span class="reply-to" v-if="replyTo">
          回复 {{ replyTo.name }}
          <button type="button" class="text-link" @click="cancelReply">取消</button>
        </span>
        <span class="spacer"></span>
        <span class="muted">{{ content.length }} / 500</span>
        <button class="btn btn-sm btn-primary" type="submit" :disabled="submitting">
          {{ submitting ? '落笔中…' : '落笔' }}
        </button>
      </div>
    </form>
    <div v-else class="comment-form guest">
      <p>登录后可在此留下按语。</p>
      <RouterLink class="btn btn-sm btn-ghost" :to="loginLink()">登录</RouterLink>
    </div>

    <div v-if="list.length" class="comment-list">
      <article v-for="c in list" :key="c.id" class="comment">
        <div class="avatar">{{ initialOf(c.user?.nickname || '史') }}</div>
        <div class="comment-main">
          <div class="comment-head">
            <b>{{ c.user?.nickname || c.user?.username || '无名氏' }}</b>
            <span class="muted">{{ timeAgo(c.createdAt) }}</span>
          </div>
          <div class="comment-body">
            <span v-if="c.replyToName" class="reply-to">回复 {{ c.replyToName }}：</span>
            <p>{{ c.content }}</p>
          </div>
          <div class="comment-foot">
            <button type="button" :class="{ on: c.liked }" @click="like(c)">
              {{ c.liked ? '已赞' : '赞' }}<span v-if="c.likeCount"> {{ c.likeCount }}</span>
            </button>
            <button v-if="auth.isLoggedIn" type="button" @click="startReply(c)">回复</button>
            <button v-if="c.canDelete" type="button" class="danger" @click="remove(c)">删除</button>
          </div>
        </div>
      </article>
    </div>
    <div v-else-if="!loading" class="empty">尚无按语。第一句，留给你。</div>

    <div v-if="list.length < total" class="load-more">
      <button class="btn btn-sm btn-ghost" :disabled="loading" @click="loadMore">
        {{ loading ? '载入中…' : '再读一些' }}
      </button>
    </div>
  </section>
</template>

<style scoped>
.comments {
  margin-top: 56px;
}

.comments-head {
  display: flex;
  align-items: baseline;
  gap: 16px;
  border-bottom: 1px solid var(--ink);
  padding-bottom: 10px;
}

.comments-head h3 {
  font-family: var(--serif);
  font-size: 20px;
  font-weight: 500;
  margin: 0;
  letter-spacing: 0.08em;
}

.comment-main {
  flex: 1;
  min-width: 0;
}

.comment-body p {
  margin: 6px 0 0;
  font-size: 14px;
  line-height: 1.95;
  color: var(--ink-2);
  white-space: pre-wrap;
  word-break: break-word;
}

.comment-foot {
  display: flex;
  gap: 16px;
  margin-top: 8px;
}

.comment-foot button {
  font-size: 11px;
  color: var(--ink-3);
  letter-spacing: 0.08em;
}

.comment-foot button.on {
  color: var(--cinnabar);
}

.comment-foot button.danger:hover {
  color: var(--cinnabar-deep);
}

.guest {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-bottom: 1px solid var(--rule-2);
}

.guest p {
  margin: 0;
  color: var(--ink-3);
  font-size: 13px;
}

.load-more {
  text-align: center;
  margin-top: 20px;
}

.spacer {
  flex: 1;
}
</style>

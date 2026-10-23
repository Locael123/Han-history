<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api'
import type { EventDetail, EventPayload } from '@/api/types'

const props = defineProps<{
  modelValue: boolean
  branchId: number
  branchName: string
  event?: EventDetail | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'saved', event: EventDetail): void
}>()

const title = ref('')
const content = ref('')
const year = ref<number | null>(new Date().getFullYear())
const month = ref<number | null>(0)
const day = ref<number | null>(0)
const location = ref('')
const figures = ref('')
const tags = ref('')
const submitting = ref(false)

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    const e = props.event
    title.value = e?.title || ''
    content.value = e?.content || ''
    year.value = e?.year || new Date().getFullYear()
    month.value = e?.month || 0
    day.value = e?.day || 0
    location.value = e?.location || ''
    figures.value = (e?.figures || []).join('，')
    tags.value = (e?.tags || []).join('，')
  },
)

function split(text: string): string[] {
  return text
    .split(/[，,、;；]/)
    .map((s) => s.trim())
    .filter(Boolean)
}

function close() {
  emit('update:modelValue', false)
}

async function submit() {
  const t = title.value.trim()
  const body = content.value.trim()
  const y = Number(year.value || 0)
  const m = Number(month.value || 0)
  const d = Number(day.value || 0)

  if (t.length < 2 || t.length > 60) return ElMessage.warning('标题需 2 至 60 字')
  if (!body) return ElMessage.warning('正文不可空白')
  if (body.length > 20000) return ElMessage.warning('正文不宜超过两万字')
  if (y < 1850 || y > 2100) return ElMessage.warning('年份应在 1850 至 2100 之间')
  if (m < 0 || m > 12) return ElMessage.warning('月份应在 0 至 12 之间')
  if (d < 0 || d > 31) return ElMessage.warning('日期应在 0 至 31 之间')
  if (d > 0 && m === 0) return ElMessage.warning('写了日，便要写月')
  if (location.value.length > 40) return ElMessage.warning('地点不宜超过 40 字')

  const fs = split(figures.value)
  const ts = split(tags.value)
  if (fs.length > 8) return ElMessage.warning('人物最多 8 位')
  if (fs.some((x) => x.length > 24)) return ElMessage.warning('单个人物名不宜超过 24 字')
  if (ts.length > 6) return ElMessage.warning('标签最多 6 个')
  if (ts.some((x) => x.length > 16)) return ElMessage.warning('单个标签不宜超过 16 字')

  const payload: EventPayload = {
    title: t,
    content: body,
    year: y,
    month: m,
    day: d,
    location: location.value.trim(),
    figures: fs,
    tags: ts,
  }

  submitting.value = true
  try {
    const saved = props.event
      ? await api.updateEvent(props.event.id, payload)
      : await api.createEvent(props.branchId, payload)
    emit('saved', saved)
    close()
    ElMessage.success(props.event ? '已改定' : '已录入')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    width="640px"
    align-center
    class="gonghe-dialog"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template #header>
      <div class="dlg-head">
        <div class="kicker">{{ event ? '改定旧稿' : '续写新条' }}</div>
        <h3>{{ event ? '改一条实录' : '记一条实录' }}</h3>
        <p class="muted">归入《{{ branchName }}》</p>
      </div>
    </template>

    <div class="field">
      <label>标题</label>
      <input v-model="title" maxlength="60" placeholder="如：某年某月，某事发生" />
    </div>

    <div class="field-row">
      <div class="field">
        <label>年</label>
        <input v-model.number="year" type="number" min="1850" max="2100" />
      </div>
      <div class="field">
        <label>月</label>
        <input v-model.number="month" type="number" min="0" max="12" />
      </div>
      <div class="field">
        <label>日</label>
        <input v-model.number="day" type="number" min="0" max="31" />
      </div>
    </div>

    <div class="field">
      <label>正文</label>
      <textarea v-model="content" rows="8" maxlength="20000" placeholder="此处可白话，可文言，可夹叙夹议。"></textarea>
      <span class="muted">{{ content.length }} / 20000</span>
    </div>

    <div class="field">
      <label>地点</label>
      <input v-model="location" maxlength="40" placeholder="如：上海·百乐门" />
    </div>
    <div class="field">
      <label>人物</label>
      <input v-model="figures" placeholder="以逗号分隔，最多 8 位" />
    </div>
    <div class="field">
      <label>标签</label>
      <input v-model="tags" placeholder="以逗号分隔，最多 6 个" />
    </div>

    <template #footer>
      <button class="btn btn-ghost btn-sm" type="button" @click="close">搁笔</button>
      <button class="btn btn-primary btn-sm" type="button" :disabled="submitting" @click="submit">
        {{ submitting ? '落墨中…' : event ? '改定' : '录入' }}
      </button>
    </template>
  </el-dialog>
</template>

<style scoped>
.dlg-head h3 {
  font-family: var(--serif);
  font-size: 26px;
  font-weight: 500;
  letter-spacing: 0.12em;
  margin: 6px 0 4px;
}

.dlg-head p {
  margin: 0;
  font-size: 12px;
}

.field input[type='number'] {
  max-width: 120px;
}
</style>

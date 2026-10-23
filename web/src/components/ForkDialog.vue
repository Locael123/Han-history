<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api'
import type { BranchDetail } from '@/api/types'

const props = defineProps<{
  modelValue: boolean
  parentId: number
  parentName: string
  forkEventId: number
  forkEventTitle: string
  forkYear: number
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'forked', branch: BranchDetail): void
}>()

const name = ref('')
const description = ref('')
const submitting = ref(false)

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      name.value = ''
      description.value = ''
    }
  },
)

function close() {
  emit('update:modelValue', false)
}

async function submit() {
  const n = name.value.trim()
  if (n.length < 2 || n.length > 40) {
    ElMessage.warning('枝名需 2 至 40 字')
    return
  }
  if (description.value.length > 300) {
    ElMessage.warning('小引不宜超过 300 字')
    return
  }
  submitting.value = true
  try {
    const branch = await api.fork({
      parentId: props.parentId,
      forkEventId: props.forkEventId,
      name: n,
      description: description.value.trim(),
    })
    emit('forked', branch)
    close()
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
    width="520px"
    align-center
    class="gonghe-dialog"
    :show-close="true"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template #header>
      <div class="dlg-head">
        <div class="kicker">自立一枝</div>
        <h3>自此分叉</h3>
      </div>
    </template>

    <div class="fork-origin">
      <p>
        自《{{ parentName }}》<template v-if="forkEventId">
          的《{{ forkEventTitle }}》（{{ forkYear }} 年）
        </template>
        <template v-else>之末（此后）</template>
        处，另开一枝。
      </p>
      <p class="muted">此前所有条目将随之承继；分叉之后，各写各的。</p>
    </div>

    <div class="field">
      <label>枝名</label>
      <input v-model="name" maxlength="40" placeholder="如：野史·城南旧事" />
      <span class="muted">{{ name.length }} / 40</span>
    </div>
    <div class="field">
      <label>小引</label>
      <textarea v-model="description" rows="3" maxlength="300" placeholder="这一枝，你想写些什么？"></textarea>
      <span class="muted">{{ description.length }} / 300</span>
    </div>

    <template #footer>
      <button class="btn btn-ghost btn-sm" type="button" @click="close">且慢</button>
      <button class="btn btn-cinnabar btn-sm" type="button" :disabled="submitting" @click="submit">
        {{ submitting ? '分叉中…' : '开枝' }}
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
  margin: 6px 0 0;
}

.fork-origin {
  border-left: 2px solid var(--cinnabar);
  padding-left: 14px;
  margin-bottom: 22px;
}

.fork-origin p {
  margin: 0;
  font-size: 13.5px;
  line-height: 1.9;
  color: var(--ink-2);
}

.fork-origin p.muted {
  margin-top: 6px;
  font-size: 12px;
}
</style>

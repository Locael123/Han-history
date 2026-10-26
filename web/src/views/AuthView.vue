<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const isRegister = computed(() => route.name === 'register')
const submitting = ref(false)

const form = reactive({
  username: '',
  nickname: '',
  password: '',
  confirm: '',
})

watch(isRegister, () => {
  form.password = ''
  form.confirm = ''
})

function switchMode(mode: 'login' | 'register') {
  router.replace({ name: mode, query: route.query })
}

function validate(): string {
  const username = form.username.trim()
  if (username.length < 3 || username.length > 32) return '名号需 3 至 32 字'
  if (!/^[a-zA-Z0-9_]+$/.test(username)) return '名号只能用字母、数字与下划线'
  if (form.password.length < 6) return '口令至少 6 位'
  if (isRegister.value) {
    if (form.nickname.trim().length < 1 || form.nickname.trim().length > 16) return '署名需 1 至 16 字'
    if (form.password !== form.confirm) return '两次口令不一致'
  }
  return ''
}

async function submit() {
  const error = validate()
  if (error) {
    ElMessage.warning(error)
    return
  }
  submitting.value = true
  try {
    if (isRegister.value) {
      await auth.register(form.username.trim(), form.password, form.nickname.trim())
      ElMessage.success('注册成功，可以开始修史了')
    } else {
      await auth.login(form.username.trim(), form.password)
      ElMessage.success(`欢迎回来，${auth.displayName}`)
    }
    const redirect = (route.query.redirect as string) || '/'
    router.replace(redirect.startsWith('/') ? redirect : '/')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    submitting.value = false
  }
}

function useDemo() {
  form.username = 'shuoshuren'
  form.password = '123456'
}
</script>

<template>
  <div class="page-wrap auth-page">
    <section class="auth-intro">
      <div class="kicker">{{ isRegister ? '注册 · 立一枝' : '登录 · 续旧稿' }}</div>
      <h2>{{ isRegister ? '留一个名号' : '接着上次写' }}</h2>
      <p class="lede">
        登录或注册，不是为了一份档案，是为了让你写下的条目有个署名，也让别人知道这枝史该由谁来续。
      </p>
      <dl class="auth-facts">
        <div>
          <dt>可做</dt>
          <dd>分叉、续写、点赞、评论</dd>
        </div>
        <div>
          <dt>不可做</dt>
          <dd>改别人的枝，替别人落笔</dd>
        </div>
        <div>
          <dt>文责</dt>
          <dd>野史各归其主，自负其言</dd>
        </div>
      </dl>
    </section>

    <section class="auth-card">
      <div class="auth-tabs">
        <button :class="{ on: !isRegister }" @click="switchMode('login')">登录</button>
        <button :class="{ on: isRegister }" @click="switchMode('register')">注册</button>
      </div>

      <form @submit.prevent="submit">
        <div class="field">
          <label>名号</label>
          <input v-model="form.username" autocomplete="username" placeholder="字母、数字、下划线" maxlength="32" />
        </div>

        <div v-if="isRegister" class="field">
          <label>署名</label>
          <input v-model="form.nickname" maxlength="16" placeholder="显示在条目与评论下" />
        </div>

        <div class="field">
          <label>口令</label>
          <input
            v-model="form.password"
            type="password"
            :autocomplete="isRegister ? 'new-password' : 'current-password'"
            placeholder="至少 6 位"
          />
        </div>

        <div v-if="isRegister" class="field">
          <label>再诵口令</label>
          <input v-model="form.confirm" type="password" autocomplete="new-password" />
        </div>

        <button class="btn btn-primary auth-submit" type="submit" :disabled="submitting">
          {{ submitting ? '请稍候…' : isRegister ? '注册' : '登录' }}
        </button>
      </form>

      <div v-if="!isRegister" class="auth-switch">
        没有名号？<button class="text-link" @click="switchMode('register')">注册</button>
        <button class="demo" type="button" @click="useDemo">试用：shuoshuren / 123456</button>
      </div>
      <div v-else class="auth-switch">
        已有名号？<button class="text-link" @click="switchMode('login')">登录</button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.auth-page {
  display: grid;
  grid-template-columns: 1fr 400px;
  gap: 80px;
  align-items: start;
  padding: 30px 0 50px;
}

.auth-intro h2 {
  font-family: var(--serif);
  font-size: clamp(34px, 4.4vw, 50px);
  font-weight: 500;
  letter-spacing: 0.1em;
  margin: 14px 0 20px;
}

.auth-facts {
  margin: 40px 0 0;
  max-width: 380px;
}

.auth-facts div {
  border-top: 1px solid var(--rule-2);
  padding: 12px 0;
}

.auth-facts dt {
  font-size: 10.5px;
  letter-spacing: 0.16em;
  color: var(--ink-3);
}

.auth-facts dd {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--ink-2);
}

.auth-card {
  border: 1px solid var(--rule);
  border-top: 2px solid var(--ink);
  background: var(--paper-2);
  padding: 28px 30px 24px;
}

.auth-tabs {
  display: flex;
  gap: 30px;
  border-bottom: 1px solid var(--rule);
  margin-bottom: 24px;
}

.auth-tabs button {
  font-family: var(--serif);
  font-size: 16px;
  font-weight: 500;
  letter-spacing: 0.18em;
  color: var(--ink-3);
  padding: 0 2px 10px;
  border-bottom: 2px solid transparent;
  transform: translateY(1px);
  transition:
    color 0.2s,
    border-color 0.2s;
}

.auth-tabs button:hover {
  color: var(--ink-2);
}

.auth-tabs button.on {
  color: var(--ink);
  border-bottom-color: var(--cinnabar);
}

.field {
  margin-bottom: 18px;
}

.field input {
  border: 0;
  border-bottom: 1px solid var(--rule);
  border-radius: 0;
  background: transparent;
  padding: 8px 2px;
  font-size: 15px;
  transition: border-color 0.2s;
}

.field input:focus {
  border-bottom-color: var(--ink);
}

.field input::placeholder {
  color: var(--ink-4);
}

.auth-submit {
  width: 100%;
  height: 44px;
  margin-top: 14px;
  border-color: var(--cinnabar);
  background: var(--cinnabar);
  color: var(--paper-2);
  font-family: var(--serif);
  font-size: 14px;
  font-weight: 500;
  letter-spacing: 0.24em;
  text-indent: 0.24em;
}

.auth-submit:hover {
  border-color: var(--ink);
  background: var(--ink);
  color: var(--paper-2);
}

.auth-switch {
  margin-top: 18px;
  font-size: 12px;
  color: var(--ink-3);
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: baseline;
}

.demo {
  flex-basis: 100%;
  margin: 8px 0 0;
  padding: 0 0 2px;
  font-family: var(--mono);
  font-size: 10px;
  letter-spacing: 0.03em;
  color: var(--ink-4);
  text-align: right;
  border-bottom: 1px dashed var(--rule-2);
}

.demo:hover {
  color: var(--cinnabar);
  border-bottom-color: var(--cinnabar);
}

@media (max-width: 900px) {
  .auth-page {
    grid-template-columns: 1fr;
    gap: 34px;
  }
}
</style>

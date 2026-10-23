<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

const today = computed(() =>
  new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' }).format(new Date()),
)

onMounted(() => {
  if (auth.isLoggedIn && !auth.me) auth.fetchMe()
})

function logout() {
  auth.logout()
  ElMessage.success('已合上自己的史稿')
  router.push('/')
}
</script>

<template>
  <header class="site-header">
    <div class="topbar">
      <div class="shell topbar-inner">
        <div class="topbar-left">
          <span>{{ today }}</span>
          <span class="dot"></span>
          <span class="edition">民间修史 · 自民国元年始</span>
        </div>
        <nav class="mainnav">
          <RouterLink to="/">卷首</RouterLink>
          <RouterLink to="/graph">谱系</RouterLink>
          <RouterLink to="/branches">史枝</RouterLink>
          <RouterLink to="/events">实录</RouterLink>
          <template v-if="auth.isLoggedIn">
            <RouterLink to="/me">{{ auth.displayName }}</RouterLink>
            <a href="javascript:void(0)" @click="logout">合卷</a>
          </template>
          <template v-else>
            <RouterLink to="/login">登录</RouterLink>
            <RouterLink to="/register">入史</RouterLink>
          </template>
        </nav>
      </div>
    </div>

    <div class="shell masthead">
      <RouterLink to="/" class="masthead-title">
        <div>
          <h1>中华共和实录</h1>
          <div class="sub">A Folk Chronicle of the Republic</div>
        </div>
        <span class="seal">野</span>
      </RouterLink>
      <div class="masthead-side">
        <div class="issue">野史卷 · 第一辑</div>
        <div class="issue-line">正史记朝廷　野史记人心</div>
        <div class="issue-line dim">凡例：可读，可续，可分叉</div>
      </div>
    </div>
    <div class="shell"><div class="double-rule"></div></div>
  </header>
</template>

<style scoped>
.masthead-side {
  text-align: right;
  padding-bottom: 6px;
}

.issue {
  font-family: var(--serif);
  font-size: 15px;
  letter-spacing: 0.22em;
  color: var(--ink-2);
}

.issue-line {
  margin-top: 6px;
  font-family: var(--sans);
  font-size: 10px;
  letter-spacing: 0.16em;
  color: var(--ink-3);
}

.issue-line.dim {
  color: var(--ink-4);
}

@media (max-width: 860px) {
  .masthead-side {
    display: none;
  }
}
</style>

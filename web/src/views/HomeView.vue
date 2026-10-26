<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '@/api'
import type { BranchCard, GraphNode, Stats, TimelineItem } from '@/api/types'
import BranchCardView from '@/components/BranchCard.vue'
import LineageGraph from '@/components/LineageGraph.vue'
import { republicYear } from '@/utils/format'

const router = useRouter()

const stats = ref<Stats | null>(null)
const official = ref<TimelineItem[]>([])
const folk = ref<TimelineItem[]>([])
const branches = ref<BranchCard[]>([])
const graph = ref<GraphNode[]>([])
const loading = ref(true)

const today = new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' }).format(new Date())
const thisYear = new Date().getFullYear()

const featured = computed(() => branches.value.filter((b) => !b.official).slice(0, 3))
const officialBranches = computed(() => branches.value.filter((b) => b.official))

onMounted(async () => {
  try {
    const [s, off, flk, br, g] = await Promise.all([
      api.stats(),
      api.events({ scope: 'official', sort: 'year', pageSize: 5 }),
      api.events({ scope: 'folk', sort: 'year', pageSize: 5 }),
      api.branches({ sort: 'hot', pageSize: 6 }),
      api.graph(),
    ])
    stats.value = s
    official.value = off.list
    folk.value = flk.list
    branches.value = br.list
    graph.value = g
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
})

function goBranch(id: number) {
  router.push(`/branches/${id}`)
}
</script>

<template>
  <div class="page-wrap home" v-loading="loading">
    <section class="hero rise">
      <div class="hero-main">
        <div class="hero-date">{{ today }}　·　{{ republicYear(thisYear) }}</div>
        <h2 class="hero-title">正史一树<br />野史千枝</h2>
        <p class="lede">
          《中华共和实录》只修一条主干。其余，由读史的人各自接续——你在哪一年不肯点头，就在哪一年另开一枝。
          同一件事，可以有无数种记法；枝与枝之间，不必相同。
        </p>
        <div class="hero-actions">
          <RouterLink class="btn btn-primary" to="/branches">读史枝总目</RouterLink>
          <RouterLink class="btn btn-ghost" to="/graph">看谱系图</RouterLink>
        </div>
      </div>
      <aside class="hero-aside">
        <div class="seal-box">
          <span class="seal">野</span>
          <p>野史者，非妄言也。<br />是不肯只留一种说法。</p>
        </div>
        <dl class="hero-facts">
          <div>
            <dt>史枝</dt>
            <dd>{{ stats?.totalBranches ?? '—' }}</dd>
          </div>
          <div>
            <dt>实录</dt>
            <dd>{{ stats?.totalEvents ?? '—' }}</dd>
          </div>
          <div>
            <dt>史官</dt>
            <dd>{{ stats?.chroniclers ?? '—' }}</dd>
          </div>
          <div>
            <dt>今日新记</dt>
            <dd>{{ stats?.todayEvents ?? 0 }}</dd>
          </div>
        </dl>
      </aside>
    </section>

    <div class="stats">
      <div class="stat">
        <b>{{ stats?.officialEvents ?? 0 }}</b>
        <span>官修条目</span>
      </div>
      <div class="stat">
        <b class="hot">{{ stats?.folkEvents ?? 0 }}</b>
        <span>野史条目</span>
      </div>
      <div class="stat">
        <b>{{ stats?.totalBranches ?? 0 }}</b>
        <span>史枝</span>
      </div>
      <div class="stat">
        <b>{{ stats?.totalLikes ?? 0 }}</b>
        <span>总赞数</span>
      </div>
    </div>

    <section class="section rise-2">
      <div class="section-head">
        <span class="ordinal">壹</span>
        <h2>官修正史</h2>
        <span class="rule"></span>
        <RouterLink class="more" to="/events?scope=official">全部条目 →</RouterLink>
      </div>
      <div class="split">
        <div class="entry-list">
          <RouterLink v-for="e in official" :key="e.id" class="entry" :to="`/events/${e.id}`">
            <div class="entry-date">
              <b>{{ e.year }}</b>
              <span>{{ e.month ? `${e.month} 月` : '年份未详' }}</span>
            </div>
            <div class="entry-main">
              <h3>{{ e.title }}</h3>
              <p>{{ e.excerpt }}</p>
              <div v-if="e.tags.length" class="entry-tags">
                <span v-for="t in e.tags.slice(0, 3)" :key="t" class="chip">#{{ t }}</span>
              </div>
            </div>
            <div class="entry-side">
              <span class="branch-link">{{ e.branchName }}</span>
            </div>
          </RouterLink>
        </div>

        <aside class="home-side">
          <div class="aside-block">
            <div class="kicker">正史主干</div>
            <RouterLink v-for="b in officialBranches" :key="b.id" :to="`/branches/${b.id}`" class="side-line">
              <span>{{ b.name }}</span>
              <em>{{ b.eventCount }} 条</em>
            </RouterLink>
          </div>
          <div class="aside-block">
            <div class="kicker">凡例</div>
            <p class="aside-note">
              一、正史为官修，只读；欲续者，自某一处分叉。<br />
              二、野史各归其主，文责自负。<br />
              三、赞与评，对事不对枝。
            </p>
          </div>
        </aside>
      </div>
    </section>

    <section class="section">
      <div class="section-head">
        <span class="ordinal">贰</span>
        <h2>野史新记</h2>
        <span class="rule"></span>
        <RouterLink class="more" to="/events?scope=folk">全部野史 →</RouterLink>
      </div>
      <div class="entry-list">
        <RouterLink v-for="e in folk" :key="e.id" class="entry" :to="`/events/${e.id}`">
          <div class="entry-date">
            <b>{{ e.year }}</b>
            <span>{{ e.month ? `${e.month} 月` : '年份未详' }}</span>
          </div>
          <div class="entry-main">
            <h3>{{ e.title }}</h3>
            <p>{{ e.excerpt }}</p>
            <div class="entry-tags">
              <RouterLink class="branch-link" :to="`/branches/${e.branchId}`" @click.stop>
                {{ e.branchName }}
              </RouterLink>
              <span v-for="t in e.tags.slice(0, 2)" :key="t" class="chip chip-red">#{{ t }}</span>
            </div>
          </div>
          <div class="entry-side">
            <span>{{ e.likeCount }} 赞</span>
          </div>
        </RouterLink>
      </div>
    </section>

    <section v-if="featured.length" class="section">
      <div class="section-head">
        <span class="ordinal">叁</span>
        <h2>史枝选刊</h2>
        <span class="rule"></span>
        <RouterLink class="more" to="/branches">史枝总目 →</RouterLink>
      </div>
      <div class="branch-grid">
        <BranchCardView v-for="b in featured" :key="b.id" :branch="b" />
      </div>
    </section>

    <section class="section">
      <div class="section-head">
        <span class="ordinal">肆</span>
        <h2>谱系</h2>
        <span class="rule"></span>
        <RouterLink class="more" to="/graph">展全图 →</RouterLink>
      </div>
      <LineageGraph :nodes="graph" @select="goBranch" />
    </section>

    <section class="cta">
      <div>
        <div class="kicker">你也可以修史</div>
        <h2>不同意，就分叉。</h2>
        <p>从任何一条已有实录处分出你的枝，写你的年份、你的人物、你的说法。</p>
      </div>
      <RouterLink class="btn btn-cinnabar" to="/branches">择一枝，另起</RouterLink>
    </section>

    <p class="margin-note">正史一树　野史千枝　可读可续可分叉</p>
  </div>
</template>

<style scoped>
.hero-main {
  max-width: 640px;
}

.home > .stats {
  grid-template-columns: repeat(4, 1fr);
  gap: 0;
  border-top: 1px solid var(--ink);
  border-bottom: 1px solid var(--rule);
}

.home > .stats .stat {
  padding: 18px 20px 18px 0;
  border-right: 1px solid var(--rule-2);
}

.home > .stats .stat:last-child {
  border-right: 0;
}

.home > .stats .stat span {
  display: block;
  margin-top: 6px;
  font-size: 10.5px;
  letter-spacing: 0.14em;
  color: var(--ink-3);
}

.hero-title {
  font-family: var(--serif);
  font-size: clamp(46px, 6.4vw, 78px);
  line-height: 1.12;
  letter-spacing: 0.06em;
  margin: 18px 0 26px;
  font-weight: 500;
}

.hero-title br {
  display: block;
}

.hero-date {
  font-family: var(--mono);
  font-size: 11px;
  letter-spacing: 0.18em;
  color: var(--ink-3);
}

.lede {
  font-size: 15.5px;
  line-height: 2.05;
  color: var(--ink-2);
}

.seal-box {
  border: 1px solid var(--rule);
  background: var(--wash);
  padding: 22px 20px;
  display: flex;
  gap: 16px;
  align-items: flex-start;
}

.seal-box p {
  font-family: var(--serif);
  font-size: 13px;
  line-height: 2;
  color: var(--ink-2);
  margin: 0;
}

.hero-facts {
  margin: 22px 0 0;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px 10px;
}

.hero-facts div {
  border-top: 1px solid var(--rule-2);
  padding-top: 8px;
}

.hero-facts dt {
  font-size: 10.5px;
  letter-spacing: 0.14em;
  color: var(--ink-3);
}

.hero-facts dd {
  margin: 4px 0 0;
  font-family: var(--serif);
  font-size: 24px;
  color: var(--ink);
}

.split {
  display: grid;
  grid-template-columns: 1fr 250px;
  gap: 48px;
  align-items: start;
}

.home-side {
  border-left: 1px solid var(--rule-2);
  padding-left: 26px;
}

.side-line {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  padding: 11px 0;
  border-bottom: 1px solid var(--rule-2);
  font-family: var(--serif);
  font-size: 14px;
  color: var(--ink-2);
}

.side-line em {
  font-style: normal;
  font-family: var(--mono);
  font-size: 10.5px;
  color: var(--ink-3);
}

.aside-note {
  font-size: 12px;
  line-height: 2.2;
  color: var(--ink-3);
  margin: 10px 0 0;
}

.cta {
  margin: 76px 0 10px;
  border-top: 1px solid var(--ink);
  border-bottom: 1px solid var(--ink);
  padding: 38px 0;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 32px;
}

.cta h2 {
  font-family: var(--serif);
  font-size: 38px;
  letter-spacing: 0.08em;
  margin: 12px 0 10px;
  font-weight: 500;
}

.cta p {
  color: var(--ink-2);
  font-size: 13.5px;
  margin: 0;
}

@media (max-width: 980px) {
  .split {
    grid-template-columns: 1fr;
    gap: 30px;
  }

  .home-side {
    border-left: 0;
    border-top: 1px solid var(--rule-2);
    padding: 22px 0 0;
  }
}

@media (max-width: 720px) {
  .cta {
    flex-direction: column;
    align-items: flex-start;
  }

  .cta h2 {
    font-size: 30px;
  }
}
</style>

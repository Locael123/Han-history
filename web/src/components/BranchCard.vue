<script setup lang="ts">
import { computed } from 'vue'
import type { BranchCard } from '@/api/types'
import { branchTone } from '@/utils/format'

const props = defineProps<{ branch: BranchCard }>()

const tone = computed(() => branchTone(props.branch.official, props.branch.id))
const ownerName = computed(() => props.branch.owner?.nickname || '史馆')
</script>

<template>
  <RouterLink
    :to="`/branches/${branch.id}`"
    class="branch-card"
    :style="{ '--branch-color': tone }"
  >
    <div class="branch-card-top">
      <h3>{{ branch.name }}</h3>
      <span :class="branch.official ? 'tag-official' : 'tag-folk'">
        {{ branch.official ? '官修' : '野史' }}
      </span>
    </div>
    <p class="desc">{{ branch.description || '此枝尚无小引。' }}</p>
    <div class="owner-line">
      <template v-if="branch.parentId">
        自《{{ branch.parentName }}》{{ branch.forkYear }} 年分叉 · {{ branch.inheritedCount }} 条承继
      </template>
      <template v-else>正史主干 · 肇于共和元年</template>
    </div>
    <div class="meta">
      <span class="strong">{{ branch.eventCount }} 条实录</span>
      <span class="sep">·</span>
      <span>{{ branch.forkCount }} 处分叉</span>
      <span class="sep">·</span>
      <span>{{ branch.likeCount }} 赞</span>
      <span class="sep">·</span>
      <span>{{ ownerName }}</span>
    </div>
  </RouterLink>
</template>

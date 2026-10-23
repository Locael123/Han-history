<template>
  <footer class="site-footer">
    <div class="shell footer-inner">
      <div>
        <div class="foot-title">中华共和实录 · 野</div>
        <p class="foot-note">
          本站所载，多为民间叙述与文学化重写，非官方史书。同一件事，人人可有一枝；枝与枝之间，不必相同。
        </p>
      </div>
      <div class="footer-links">
        <RouterLink to="/branches">史枝总目</RouterLink>
        <RouterLink to="/graph">谱系图</RouterLink>
        <RouterLink to="/events">全站实录</RouterLink>
        <RouterLink to="/register">立一枝</RouterLink>
      </div>
      <div class="footer-meta">
        <div>野史卷 · 第一辑</div>
        <div>二〇二六 · 民间修史</div>
      </div>
    </div>
  </footer>
</template>

<style scoped>
.footer-meta {
  text-align: right;
  line-height: 2;
}

@media (max-width: 720px) {
  .footer-meta {
    text-align: left;
  }
}
</style>

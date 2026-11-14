<script setup lang="ts">
import {
  DEFAULT_TYPOGRAPHY,
  OASIS_HEIGHT_MM,
  OASIS_HEIGHT_PT,
  OASIS_WIDTH_MM,
  OASIS_WIDTH_PT,
  type OasisPage,
} from '../lib/paginator'
import type { StreamStatus } from '../lib/useTranslateSSE'

defineProps<{
  pages: OasisPage[]
  status: StreamStatus
  marginHorizontalPt?: number
}>()

const scale = 1.28 // Scales 303.36pt × 403.20pt up smoothly for crisp desktop inspection
const marginH = DEFAULT_TYPOGRAPHY.marginHorizontalPt
</script>

<template>
  <section class="oasis-grid-section">
    <div class="grid-header">
      <div class="spec-pill">
        Kindle Oasis 7" Display Geometry:
        <strong>{{ OASIS_WIDTH_MM }} × {{ OASIS_HEIGHT_MM }} mm</strong>
        ({{ OASIS_WIDTH_PT }} × {{ OASIS_HEIGHT_PT }} pt • 3:4 @ 300 PPI)
      </div>
      <div class="legend">
        <span class="legend-item">
          <span class="swatch swatch-source" />
          Paragraph 1: Source Text (Serif)
        </span>
        <span class="legend-item">
          <span class="swatch swatch-translation" />
          Paragraph 2: Translation (Italic + Left Bar)
        </span>
      </div>
    </div>

    <div v-if="pages.length === 0" class="empty-state">
      <div class="empty-bezel">
        <div class="empty-screen">
          <p class="empty-title">Kindle Oasis Live Page Preview</p>
          <p class="empty-desc">
            Drop or select a text above. Each paragraph will be translated and progressively laid out onto 107.02 × 142.24 mm pages here.
          </p>
        </div>
      </div>
    </div>

    <div v-else class="pages-grid">
      <article
        v-for="page in pages"
        :key="page.pageNumber"
        class="oasis-device-card"
      >
        <div class="device-header">
          <span class="page-badge">Page {{ page.pageNumber }} of {{ pages.length }}</span>
          <span class="dimensions-tag">107.02 × 142.24 mm</span>
        </div>

        <!-- Exact 303.36pt × 403.20pt coordinate space scaled proportionally -->
        <div
          class="oasis-screen-viewport"
          :style="{
            width: `${OASIS_WIDTH_PT * scale}px`,
            height: `${OASIS_HEIGHT_PT * scale}px`,
          }"
        >
          <div
            class="oasis-page-canvas"
            :style="{
              width: `${OASIS_WIDTH_PT}px`,
              height: `${OASIS_HEIGHT_PT}px`,
              transform: `scale(${scale})`,
              transformOrigin: 'top left',
            }"
          >
            <div
              v-for="(block, bIdx) in page.blocks"
              :key="`${page.pageNumber}-${bIdx}`"
              class="page-block"
              :class="[`block-${block.kind}`]"
              :style="{
                top: `${block.yStartPt}px`,
                left: `${(marginHorizontalPt ?? marginH) + block.indentPt}px`,
                right: `${marginHorizontalPt ?? marginH}px`,
                fontSize: `${block.fontSizePt}px`,
                lineHeight: `${block.lineHeightPt}px`,
              }"
            >
              <div
                v-if="block.kind === 'translation'"
                class="translation-bar"
                :style="{
                  left: `-${block.indentPt - 2.5}px`,
                  height: `${block.heightPt - 3}px`,
                }"
              />
              <div
                v-for="(line, lIdx) in block.lines"
                :key="lIdx"
                class="text-line"
                :style="{ height: `${block.lineHeightPt}px` }"
              >
                {{ line }}
              </div>
            </div>

            <footer class="oasis-page-footer">
              {{ page.pageNumber }} / {{ pages.length }}
            </footer>
          </div>
        </div>
      </article>

      <div v-if="status === 'translating' || status === 'detecting'" class="streaming-card">
        <div class="spinner" />
        <p>Streaming next paragraph translation & paginating…</p>
      </div>
    </div>
  </section>
</template>

<style scoped>
.oasis-grid-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.grid-header {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 0.75rem;
  font-size: 0.82rem;
  color: #5b564e;
}

.spec-pill {
  padding: 0.35rem 0.75rem;
  background: #efece4;
  border-radius: 6px;
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
}

.swatch {
  width: 12px;
  height: 12px;
  border-radius: 3px;
}

.swatch-source {
  background: #000000;
}

.swatch-translation {
  background: #4b4b4b;
  border-left: 3px solid #1c1b18;
}

.empty-state {
  display: flex;
  justify-content: center;
  padding: 2rem 0;
}

.empty-bezel {
  padding: 18px 28px 18px 18px;
  background: #232220;
  border-radius: 18px;
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.14);
}

.empty-screen {
  width: 303px;
  height: 403px;
  background: #f6f4ee;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding: 2rem;
  box-sizing: border-box;
  text-align: center;
  font-family: 'Literata', Georgia, serif;
}

.empty-title {
  font-size: 1.05rem;
  font-weight: 700;
  color: #1c1b18;
  margin: 0 0 0.5rem;
}

.empty-desc {
  font-size: 0.86rem;
  color: #5f5a52;
  line-height: 1.5;
  margin: 0;
}

.pages-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 1.5rem;
  align-items: flex-start;
}

.oasis-device-card {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  animation: pageFadeIn 0.28s ease-out;
}

@keyframes pageFadeIn {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.device-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.78rem;
  color: #575249;
  padding: 0 0.25rem;
}

.page-badge {
  font-weight: 700;
  color: #1c1b18;
}

.dimensions-tag {
  font-family: monospace;
  color: #787268;
}

.oasis-screen-viewport {
  background: #f8f6f0;
  border: 10px solid #262523;
  border-right-width: 22px; /* Asymmetric Kindle Oasis ergonomic grip bezel */
  border-radius: 14px;
  box-shadow: 0 8px 22px rgba(20, 20, 18, 0.12);
  overflow: hidden;
}

.oasis-page-canvas {
  position: relative;
  background: #f8f6f0;
  font-family: 'Literata', Georgia, serif;
  color: #000000;
  user-select: text;
}

.page-block {
  position: absolute;
}

.block-source {
  color: #000000;
  font-style: normal;
}

.block-translation {
  color: #2e2e2e;
  font-style: italic;
}

.translation-bar {
  position: absolute;
  top: 2px;
  width: 1.4px;
  background: #6b6b6b;
}

.text-line {
  white-space: pre;
  overflow: hidden;
}

.oasis-page-footer {
  position: absolute;
  bottom: 7px;
  left: 0;
  right: 0;
  text-align: center;
  font-size: 7.5px;
  color: #737067;
  font-family: 'Literata', Georgia, serif;
}

.streaming-card {
  width: 220px;
  min-height: 160px;
  border: 1px dashed #c8c2b4;
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding: 1.25rem;
  text-align: center;
  color: #5b564e;
  font-size: 0.85rem;
  gap: 0.75rem;
}

.spinner {
  width: 22px;
  height: 22px;
  border: 2.5px solid #d9d4c7;
  border-top-color: #1c1b18;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>

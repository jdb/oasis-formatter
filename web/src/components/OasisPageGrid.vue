<script setup lang="ts">
import { ref } from 'vue'
import {
  DEFAULT_TYPOGRAPHY,
  OASIS_HEIGHT_MM,
  OASIS_HEIGHT_PT,
  OASIS_WIDTH_MM,
  OASIS_WIDTH_PT,
  type OasisPage,
} from '../lib/paginator'
import type { LanguageInfo, StreamStatus } from '../lib/useTranslateSSE'

defineProps<{
  hasText: boolean
  pages: OasisPage[]
  status: StreamStatus
  translateEnabled: boolean
  languageInfo: LanguageInfo | null
  translatedCount: number
  totalParagraphs: number
  isExportingPdf: boolean
  marginHorizontalPt?: number
}>()

const emit = defineEmits<{
  (e: 'run'): void
  (e: 'stop'): void
  (e: 'reset'): void
  (e: 'export-pdf'): void
}>()

const showMoreInfo = ref(false)
const scale = 1.24
const marginH = DEFAULT_TYPOGRAPHY.marginHorizontalPt
</script>

<template>
  <section class="step-card">
    <div class="step-header">
      <div class="title-group">
        <button
          type="button"
          class="btn-primary btn-cta"
          :disabled="!hasText || status === 'translating' || status === 'detecting'"
          @click="emit('run')"
        >
          {{
            status === 'translating' || status === 'detecting'
              ? 'Formatting…'
              : translateEnabled
                ? 'Format, with translation'
                : 'Format'
          }}
        </button>

        <span v-if="status === 'detecting'" class="status-text">Detecting language…</span>
        <span v-else-if="status === 'translating' && languageInfo" class="status-text">
          {{ languageInfo.sourceLangName }} → {{ languageInfo.targetLangName }}
          ({{ translatedCount }}/{{ totalParagraphs }})
        </span>
        <span v-else-if="translateEnabled && languageInfo && totalParagraphs > 0" class="status-text">
          {{ languageInfo.sourceLangName }} → {{ languageInfo.targetLangName }}
          ({{ translatedCount }} paragraphs, {{ pages.length }} {{ pages.length === 1 ? 'page' : 'pages' }})
        </span>
        <span v-else-if="!translateEnabled && pages.length > 0" class="status-text">
          {{ totalParagraphs }} paragraphs, {{ pages.length }} {{ pages.length === 1 ? 'page' : 'pages' }}
        </span>
      </div>

      <div class="header-actions">
        <button
          v-if="status === 'translating' || status === 'detecting'"
          type="button"
          class="btn-subtle"
          @click="emit('stop')"
        >
          Pause
        </button>

        <button
          v-if="pages.length > 0"
          type="button"
          class="btn-subtle"
          :disabled="isExportingPdf"
          @click="emit('export-pdf')"
        >
          {{ isExportingPdf ? 'Generating PDF…' : 'Download PDF' }}
        </button>

        <button
          type="button"
          class="btn-subtle"
          @click="showMoreInfo = !showMoreInfo"
        >
          {{ showMoreInfo ? 'Hide info' : 'Show more info' }}
        </button>

        <button
          v-if="hasText || pages.length > 0"
          type="button"
          class="btn-subtle"
          @click="emit('reset')"
        >
          Clear
        </button>
      </div>
    </div>

    <div v-if="showMoreInfo" class="more-info">
      <p class="info-line">
        Kindle Oasis 7" Display Geometry: {{ OASIS_WIDTH_MM }} × {{ OASIS_HEIGHT_MM }} mm
        ({{ OASIS_WIDTH_PT }} × {{ OASIS_HEIGHT_PT }} pt • 3:4 @ 300 PPI)
      </p>
      <p v-if="translateEnabled" class="info-line">
        Each pair renders the source paragraph first, followed by its translation (italic, indented).
      </p>
    </div>

    <div v-if="pages.length === 0" class="empty-preview">
      Preview pages will appear here.
    </div>

    <div v-else class="pages-grid">
      <article
        v-for="page in pages"
        :key="page.pageNumber"
        class="oasis-page-item"
      >
        <div class="page-caption">
          Page {{ page.pageNumber }} / {{ pages.length }}
        </div>

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

      <div v-if="status === 'translating' || status === 'detecting'" class="streaming-note">
        Streaming next paragraph…
      </div>
    </div>
  </section>
</template>

<style scoped>
.step-header {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.85rem;
}

.title-group {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
}

.btn-cta {
  padding: 0.55rem 1.15rem;
  font-size: 0.94rem;
}

.status-text {
  font-size: 0.86rem;
  color: #6b665e;
}

.header-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.more-info {
  margin-bottom: 1rem;
  padding: 0.65rem 0.85rem;
  background: #f7f5f0;
  border-radius: 6px;
  font-size: 0.84rem;
  color: #5b564e;
}

.info-line {
  margin: 0;
  line-height: 1.45;
}

.info-line + .info-line {
  margin-top: 0.25rem;
}

.empty-preview {
  padding: 2.5rem 1rem;
  text-align: center;
  font-size: 0.9rem;
  color: #6b665e;
}

.pages-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 1.25rem;
  align-items: flex-start;
}

.oasis-page-item {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.page-caption {
  font-size: 0.8rem;
  color: #6b665e;
}

.oasis-screen-viewport {
  background: #f8f6f0;
  border: 1px solid #d6d2c8;
  border-radius: 4px;
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
  width: 1.2px;
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

.streaming-note {
  padding: 1rem;
  font-size: 0.88rem;
  color: #6b665e;
  align-self: center;
}
</style>

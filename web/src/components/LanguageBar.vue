<script setup lang="ts">
import type { LanguageInfo, StreamStatus } from '../lib/useTranslateSSE'

export interface LanguageOption {
  code: string
  name: string
}

defineProps<{
  languages: LanguageOption[]
  targetLang: string
  sourceFontSizePt: number
  status: StreamStatus
  languageInfo: LanguageInfo | null
  translatedCount: number
  totalParagraphs: number
  pageCount: number
  isExportingPdf: boolean
}>()

const emit = defineEmits<{
  (e: 'update:targetLang', code: string): void
  (e: 'update:sourceFontSizePt', size: number): void
  (e: 'export-pdf'): void
  (e: 'stop'): void
  (e: 'reset'): void
}>()
</script>

<template>
  <section class="language-bar">
    <div class="controls-group">
      <label class="field">
        <span class="field-label">Destination Language</span>
        <select
          class="select-input"
          :value="targetLang"
          @change="emit('update:targetLang', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="lang in languages" :key="lang.code" :value="lang.code">
            {{ lang.name }} ({{ lang.code.toUpperCase() }})
          </option>
        </select>
      </label>

      <label class="field">
        <span class="field-label">Oasis Font Size</span>
        <select
          class="select-input"
          :value="sourceFontSizePt"
          @change="emit('update:sourceFontSizePt', Number(($event.target as HTMLSelectElement).value))"
        >
          <option :value="9.5">Compact (9.5 pt)</option>
          <option :value="10.5">Standard (10.5 pt)</option>
          <option :value="11">Comfort (11 pt)</option>
          <option :value="12">Large (12 pt)</option>
          <option :value="13">Extra Large (13 pt)</option>
        </select>
      </label>
    </div>

    <div class="status-group">
      <div v-if="languageInfo" class="detected-badge">
        <span class="dot" :class="{ pulse: status === 'translating' || status === 'detecting' }" />
        <span>
          Detected: <strong>{{ languageInfo.sourceLangName }}</strong>
          ({{ languageInfo.sourceLang.toUpperCase() }})
          → <strong>{{ languageInfo.targetLangName }}</strong>
        </span>
      </div>
      <div v-else-if="status === 'detecting'" class="detected-badge">
        <span class="dot pulse" />
        <span>Detecting source language…</span>
      </div>

      <div v-if="totalParagraphs > 0" class="progress-pill">
        <span>{{ translatedCount }} / {{ totalParagraphs }} paragraphs</span>
        <span class="sep">•</span>
        <strong>{{ pageCount }} {{ pageCount === 1 ? 'Oasis page' : 'Oasis pages' }}</strong>
      </div>
    </div>

    <div class="actions-group">
      <button
        v-if="status === 'translating' || status === 'detecting'"
        type="button"
        class="btn btn-secondary"
        @click="emit('stop')"
      >
        Pause Stream
      </button>

      <button
        v-if="pageCount > 0"
        type="button"
        class="btn btn-primary"
        :disabled="isExportingPdf"
        @click="emit('export-pdf')"
      >
        {{ isExportingPdf ? 'Generating PDF…' : 'Download Kindle Oasis PDF (107×142mm)' }}
      </button>

      <button
        v-if="totalParagraphs > 0"
        type="button"
        class="btn btn-ghost"
        @click="emit('reset')"
      >
        Clear
      </button>
    </div>
  </section>
</template>

<style scoped>
.language-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.9rem 1.25rem;
  background: #ffffff;
  border: 1px solid #e3e0d8;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(20, 20, 19, 0.04);
}

.controls-group,
.status-group,
.actions-group {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.field-label {
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #6b665e;
}

.select-input {
  padding: 0.45rem 0.75rem;
  font-size: 0.9rem;
  border: 1px solid #d2cec4;
  border-radius: 8px;
  background: #faf9f5;
  color: #1c1b18;
  font-weight: 500;
  cursor: pointer;
}

.detected-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 0.8rem;
  background: #f1efe8;
  border: 1px solid #dcd8cc;
  border-radius: 999px;
  font-size: 0.85rem;
  color: #2c2a25;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #2e7d32;
}

.dot.pulse {
  background: #d97706;
  animation: pulseAnim 1.1s infinite ease-in-out;
}

@keyframes pulseAnim {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.35); opacity: 0.55; }
}

.progress-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.85rem;
  color: #4b473f;
  padding: 0.35rem 0.7rem;
  background: #faf8f3;
  border-radius: 8px;
}

.sep {
  color: #b5b0a4;
}

.btn {
  padding: 0.55rem 1rem;
  border-radius: 8px;
  font-size: 0.88rem;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid transparent;
  transition: all 0.15s ease;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: #1c1b18;
  color: #faf9f5;
}

.btn-primary:hover:not(:disabled) {
  background: #33312b;
}

.btn-secondary {
  background: #fff8eb;
  border-color: #f59e0b;
  color: #92400e;
}

.btn-ghost {
  background: transparent;
  border-color: #d6d2c8;
  color: #57534a;
}

.btn-ghost:hover {
  background: #f3f1ea;
}
</style>

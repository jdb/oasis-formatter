<script setup lang="ts">
import type { LanguageInfo, StreamStatus } from '../lib/useTranslateSSE'

export interface LanguageOption {
  code: string
  name: string
}

defineProps<{
  hasText: boolean
  translateEnabled: boolean
  languages: LanguageOption[]
  targetLang: string
  sourceFontSizePt: number
  status: StreamStatus
  languageInfo: LanguageInfo | null
  translatedCount: number
  totalParagraphs: number
  pageCount: number
}>()

const emit = defineEmits<{
  (e: 'update:translateEnabled', enabled: boolean): void
  (e: 'update:targetLang', code: string): void
  (e: 'update:sourceFontSizePt', size: number): void
  (e: 'run'): void
  (e: 'stop'): void
  (e: 'reset'): void
}>()
</script>

<template>
  <section class="step-card">
    <div class="step-header">
      <h2 class="step-title">2. Translation &amp; formatting</h2>

      <div v-if="status === 'detecting' || status === 'translating'" class="status-text">
        <span v-if="languageInfo">
          {{ languageInfo.sourceLangName }} → {{ languageInfo.targetLangName }}
          ({{ translatedCount }}/{{ totalParagraphs }})
        </span>
        <span v-else>Detecting language…</span>
      </div>
      <div v-else-if="translateEnabled && languageInfo && totalParagraphs > 0" class="status-text">
        {{ languageInfo.sourceLangName }} → {{ languageInfo.targetLangName }}
        ({{ translatedCount }} paragraphs, {{ pageCount }} {{ pageCount === 1 ? 'page' : 'pages' }})
      </div>
      <div v-else-if="!translateEnabled && pageCount > 0" class="status-text">
        Format only ({{ totalParagraphs }} paragraphs, {{ pageCount }} {{ pageCount === 1 ? 'page' : 'pages' }})
      </div>
    </div>

    <div class="options-row">
      <label class="checkbox-field">
        <input
          type="checkbox"
          :checked="translateEnabled"
          @change="emit('update:translateEnabled', ($event.target as HTMLInputElement).checked)"
        />
        <span>Translate paragraphs</span>
      </label>

      <label v-if="translateEnabled" class="field">
        <span class="field-label">Destination language</span>
        <select
          class="select-input"
          :value="targetLang"
          @change="emit('update:targetLang', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="lang in languages" :key="lang.code" :value="lang.code">
            {{ lang.name }}
          </option>
        </select>
      </label>

      <label class="field">
        <span class="field-label">Font size</span>
        <select
          class="select-input"
          :value="sourceFontSizePt"
          @change="emit('update:sourceFontSizePt', Number(($event.target as HTMLSelectElement).value))"
        >
          <option :value="9.5">9.5 pt</option>
          <option :value="10.5">10.5 pt</option>
          <option :value="11">11 pt</option>
          <option :value="12">12 pt</option>
          <option :value="13">13 pt</option>
        </select>
      </label>

      <div class="actions">
        <button
          v-if="status === 'translating' || status === 'detecting'"
          type="button"
          class="btn-subtle"
          @click="emit('stop')"
        >
          Pause
        </button>

        <button
          type="button"
          class="btn-primary"
          :disabled="!hasText || status === 'translating' || status === 'detecting'"
          @click="emit('run')"
        >
          {{ translateEnabled ? 'Translate & format' : 'Format for Oasis' }}
        </button>

        <button
          v-if="hasText || pageCount > 0"
          type="button"
          class="btn-subtle"
          @click="emit('reset')"
        >
          Clear
        </button>
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

.status-text {
  font-size: 0.88rem;
  color: #6b665e;
}

.options-row {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 1.25rem;
}

.checkbox-field {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding-bottom: 0.45rem;
  font-size: 0.92rem;
  color: #1c1b18;
  cursor: pointer;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.field-label {
  font-size: 0.82rem;
  color: #6b665e;
}

.select-input {
  padding: 0.45rem 0.7rem;
  font-family: inherit;
  font-size: 0.9rem;
  border: 1px solid #d6d2c8;
  border-radius: 6px;
  background: #fcfbf9;
  color: #1c1b18;
  cursor: pointer;
}

.actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-left: auto;
}
</style>

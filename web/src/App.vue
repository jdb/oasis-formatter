<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import DropZone from './components/DropZone.vue'
import LanguageBar, { type LanguageOption } from './components/LanguageBar.vue'
import OasisPageGrid from './components/OasisPageGrid.vue'
import {
  createBrowserTextMeasurer,
  paginateForKindleOasis,
  type ParagraphPair,
} from './lib/paginator'
import { generateKindleOasisPDF, triggerPDFDownload } from './lib/pdfExporter'
import { useTranslateSSE } from './lib/useTranslateSSE'

const defaultLanguages: LanguageOption[] = [
  { code: 'en', name: 'English' },
  { code: 'fr', name: 'French' },
  { code: 'es', name: 'Spanish' },
  { code: 'de', name: 'German' },
  { code: 'it', name: 'Italian' },
  { code: 'pt', name: 'Portuguese' },
  { code: 'nl', name: 'Dutch' },
  { code: 'ru', name: 'Russian' },
  { code: 'ja', name: 'Japanese' },
  { code: 'zh', name: 'Chinese' },
  { code: 'pl', name: 'Polish' },
  { code: 'sv', name: 'Swedish' },
  { code: 'ko', name: 'Korean' },
  { code: 'ar', name: 'Arabic' },
]

const languages = ref<LanguageOption[]>(defaultLanguages)
const rawText = ref<string>('')
const targetLang = ref<string>('')
const translateEnabled = computed(() => targetLang.value !== '')
const sourceFontSizePt = ref<number>(11)
const isExportingPdf = ref<boolean>(false)
const formatOnlyPairs = ref<ParagraphPair[]>([])

const measurer = createBrowserTextMeasurer()

const {
  status,
  totalParagraphs,
  translatedPairs,
  languageInfo,
  errorMessage,
  startTranslationStream,
  stopStream,
  reset,
} = useTranslateSSE()

onMounted(async () => {
  try {
    const resp = await fetch('/api/languages')
    if (resp.ok) {
      const data = await resp.json()
      if (Array.isArray(data?.languages) && data.languages.length > 0) {
        languages.value = data.languages
      }
    }
  } catch {
    // fallback to defaultLanguages
  }
})

function splitParagraphsLocal(text: string): ParagraphPair[] {
  const normalized = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n').trim()
  if (!normalized) return []
  let blocks = normalized.split(/\n\s*\n+/)
  if (blocks.length === 1 && normalized.includes('\n')) {
    blocks = normalized.split('\n')
  }
  return blocks
    .map((b) => b.replace(/\s+/g, ' ').trim())
    .filter(Boolean)
    .map((source, index) => ({ index, source, translation: '' }))
}

const activePairs = computed(() =>
  translateEnabled.value ? translatedPairs.value : formatOnlyPairs.value,
)

const activeTotalParagraphs = computed(() =>
  translateEnabled.value ? totalParagraphs.value : formatOnlyPairs.value.length,
)

const typographyOpts = computed(() => ({
  sourceFontSizePt: sourceFontSizePt.value,
  translationFontSizePt: +(sourceFontSizePt.value - 1.0).toFixed(1),
}))

const oasisPages = computed(() =>
  paginateForKindleOasis(activePairs.value, typographyOpts.value, measurer),
)

function runProcessing() {
  const text = rawText.value.trim()
  if (!text) return
  if (translateEnabled.value) {
    formatOnlyPairs.value = []
    startTranslationStream(text, targetLang.value, 160)
  } else {
    stopStream()
    formatOnlyPairs.value = splitParagraphsLocal(text)
  }
}

function handleReset() {
  rawText.value = ''
  formatOnlyPairs.value = []
  reset()
}

async function handleExportPDF() {
  if (oasisPages.value.length === 0) return
  isExportingPdf.value = true
  try {
    const filename = translateEnabled.value
      ? `oasis-bilingual-${languageInfo.value?.sourceLang || 'src'}-${targetLang.value}.pdf`
      : 'oasis-formatted.pdf'
    const bytes = await generateKindleOasisPDF(oasisPages.value, typographyOpts.value, {
      title: translateEnabled.value
        ? `Oasis Formatter (${languageInfo.value?.sourceLang?.toUpperCase() || 'AUTO'} → ${targetLang.value.toUpperCase()})`
        : 'Oasis Formatter',
      sourceLangName: languageInfo.value?.sourceLangName,
      targetLangName: languageInfo.value?.targetLangName,
    })
    triggerPDFDownload(bytes, filename)
  } finally {
    isExportingPdf.value = false
  }
}
</script>

<template>
  <div class="app-shell">
    <header class="top-header">
      <h1 class="brand-title">Oasis Formatter</h1>
    </header>

    <main class="main-content">
      <DropZone v-model="rawText" />

      <LanguageBar
        :languages="languages"
        :target-lang="targetLang"
        :source-font-size-pt="sourceFontSizePt"
        @update:target-lang="targetLang = $event"
        @update:source-font-size-pt="sourceFontSizePt = $event"
      />

      <div v-if="errorMessage" class="error-banner">
        {{ errorMessage }}
      </div>

      <OasisPageGrid
        :has-text="rawText.trim().length > 0"
        :pages="oasisPages"
        :status="status"
        :translate-enabled="translateEnabled"
        :language-info="languageInfo"
        :translated-count="activePairs.length"
        :total-paragraphs="activeTotalParagraphs"
        :is-exporting-pdf="isExportingPdf"
        @run="runProcessing"
        @stop="stopStream"
        @reset="handleReset"
        @export-pdf="handleExportPDF"
      />
    </main>
  </div>
</template>

<style>
@import './assets/fonts/fonts.css';

:root {
  color-scheme: light;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
  background-color: #f4f1ea;
  color: #1c1b18;
}

body {
  margin: 0;
  padding: 0;
  background-color: #f4f1ea;
}

.app-shell {
  max-width: 1120px;
  margin: 0 auto;
  padding: 1.5rem 1.25rem 3rem;
}

.top-header {
  margin-bottom: 1rem;
}

.brand-title {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
  color: #1c1b18;
}

.main-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.step-card {
  background: #ffffff;
  border: 1px solid #e2dfd7;
  border-radius: 10px;
  padding: 1.15rem 1.25rem;
}

.step-title {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: #1c1b18;
}

.btn-primary {
  padding: 0.48rem 0.95rem;
  border-radius: 6px;
  border: 1px solid #1c1b18;
  background: #1c1b18;
  color: #faf9f5;
  font-family: inherit;
  font-size: 0.88rem;
  font-weight: 500;
  cursor: pointer;
}

.btn-primary:hover:not(:disabled) {
  background: #33312b;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-subtle {
  display: inline-flex;
  align-items: center;
  padding: 0.45rem 0.85rem;
  border-radius: 6px;
  border: 1px solid #d6d2c8;
  background: #faf9f5;
  color: #1c1b18;
  font-family: inherit;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
}

.btn-subtle:hover {
  background: #f0ede5;
}

.error-banner {
  padding: 0.7rem 1rem;
  background: #fef2f2;
  border: 1px solid #e5b4b4;
  border-radius: 8px;
  color: #8b2626;
  font-size: 0.88rem;
}
</style>

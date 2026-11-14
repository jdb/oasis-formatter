<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import DropZone from './components/DropZone.vue'
import LanguageBar, { type LanguageOption } from './components/LanguageBar.vue'
import OasisPageGrid from './components/OasisPageGrid.vue'
import {
  createBrowserTextMeasurer,
  paginateForKindleOasis,
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
const targetLang = ref<string>('en')
const sourceFontSizePt = ref<number>(11)
const isExportingPdf = ref<boolean>(false)
const lastSubmittedText = ref<string>('')

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

const typographyOpts = computed(() => ({
  sourceFontSizePt: sourceFontSizePt.value,
  translationFontSizePt: +(sourceFontSizePt.value - 1.0).toFixed(1),
}))

const oasisPages = computed(() =>
  paginateForKindleOasis(translatedPairs.value, typographyOpts.value, measurer),
)

function handleTextSubmission(text: string) {
  lastSubmittedText.value = text
  startTranslationStream(text, targetLang.value, 160)
}

function handleTargetLangChange(newLang: string) {
  targetLang.value = newLang
  if (lastSubmittedText.value.trim()) {
    startTranslationStream(lastSubmittedText.value, newLang, 160)
  }
}

async function handleExportPDF() {
  if (oasisPages.value.length === 0) return
  isExportingPdf.value = true
  try {
    const bytes = await generateKindleOasisPDF(oasisPages.value, typographyOpts.value, {
      title: `Oasis Formatter (${languageInfo.value?.sourceLang?.toUpperCase() || ' AUTO'} → ${targetLang.value.toUpperCase()})`,
      sourceLangName: languageInfo.value?.sourceLangName,
      targetLangName: languageInfo.value?.targetLangName,
    })
    const srcSlug = languageInfo.value?.sourceLang || 'src'
    triggerPDFDownload(bytes, `oasis-bilingual-${srcSlug}-${targetLang.value}.pdf`)
  } finally {
    isExportingPdf.value = false
  }
}
</script>

<template>
  <div class="app-shell">
    <header class="top-header">
      <div class="brand">
        <span class="brand-icon">📖</span>
        <div>
          <h1 class="brand-title">Oasis Formatter</h1>
          <p class="brand-tagline">
            Bilingual paragraph-by-paragraph translator &amp; 1:1 Kindle Oasis (107.02 × 142.24 mm) PDF generator
          </p>
        </div>
      </div>
    </header>

    <main class="main-content">
      <LanguageBar
        :languages="languages"
        :target-lang="targetLang"
        :source-font-size-pt="sourceFontSizePt"
        :status="status"
        :language-info="languageInfo"
        :translated-count="translatedPairs.length"
        :total-paragraphs="totalParagraphs"
        :page-count="oasisPages.length"
        :is-exporting-pdf="isExportingPdf"
        @update:target-lang="handleTargetLangChange"
        @update:source-font-size-pt="sourceFontSizePt = $event"
        @export-pdf="handleExportPDF"
        @stop="stopStream"
        @reset="reset"
      />

      <div v-if="errorMessage" class="error-banner">
        {{ errorMessage }}
      </div>

      <DropZone
        :is-busy="status === 'detecting' || status === 'translating'"
        @submit-text="handleTextSubmission"
      />

      <OasisPageGrid
        :pages="oasisPages"
        :status="status"
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
  max-width: 1280px;
  margin: 0 auto;
  padding: 1.5rem 1.25rem 3rem;
}

.top-header {
  margin-bottom: 1.25rem;
}

.brand {
  display: flex;
  align-items: center;
  gap: 0.85rem;
}

.brand-icon {
  font-size: 2rem;
}

.brand-title {
  margin: 0;
  font-size: 1.45rem;
  font-weight: 800;
  letter-spacing: -0.02em;
}

.brand-tagline {
  margin: 0.15rem 0 0;
  font-size: 0.88rem;
  color: #5e5950;
}

.main-content {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.error-banner {
  padding: 0.75rem 1rem;
  background: #fef2f2;
  border: 1px solid #f87171;
  border-radius: 8px;
  color: #991b1b;
  font-size: 0.88rem;
}
</style>

import { ref } from 'vue'
import type { ParagraphPair } from './paginator'

export type StreamStatus = 'idle' | 'detecting' | 'translating' | 'completed' | 'error'

export interface LanguageInfo {
  sourceLang: string
  sourceLangName: string
  confidence: number
  targetLang: string
  targetLangName: string
}

export function useTranslateSSE() {
  const status = ref<StreamStatus>('idle')
  const totalParagraphs = ref<number>(0)
  const rawParagraphs = ref<string[]>([])
  const translatedPairs = ref<ParagraphPair[]>([])
  const languageInfo = ref<LanguageInfo | null>(null)
  const errorMessage = ref<string | null>(null)

  let activeAbortController: AbortController | null = null

  function stopStream() {
    if (activeAbortController) {
      activeAbortController.abort()
      activeAbortController = null
    }
    if (status.value === 'detecting' || status.value === 'translating') {
      status.value = 'completed'
    }
  }

  function reset() {
    stopStream()
    status.value = 'idle'
    totalParagraphs.value = 0
    rawParagraphs.value = []
    translatedPairs.value = []
    languageInfo.value = null
    errorMessage.value = null
  }

  async function startTranslationStream(
    text: string,
    targetLang: string,
    delayMs = 160,
    onPairReceived?: (pair: ParagraphPair) => void,
  ) {
    stopStream()
    const trimmed = text.trim()
    if (!trimmed) return

    status.value = 'detecting'
    totalParagraphs.value = 0
    rawParagraphs.value = []
    translatedPairs.value = []
    languageInfo.value = null
    errorMessage.value = null

    const controller = new AbortController()
    activeAbortController = controller

    try {
      const response = await fetch('/api/translate/stream', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'text/event-stream',
        },
        body: JSON.stringify({
          text: trimmed,
          targetLang,
          delayMs,
        }),
        signal: controller.signal,
      })

      if (!response.ok || !response.body) {
        let msg = `Translation failed (${response.status})`
        try {
          const errJson = await response.json()
          if (errJson?.detail || errJson?.title) {
            msg = errJson.detail || errJson.title
          }
        } catch {
          // ignore
        }
        throw new Error(msg)
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { value, done } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const events = buffer.split('\n\n')
        buffer = events.pop() || ''

        for (const rawEvent of events) {
          const lines = rawEvent.split('\n')
          let eventName = 'message'
          let dataPayload = ''

          for (const line of lines) {
            if (line.startsWith('event:')) {
              eventName = line.slice(6).trim()
            } else if (line.startsWith('data:')) {
              dataPayload += line.slice(5).trim()
            }
          }

          if (!dataPayload) continue
          const parsed = JSON.parse(dataPayload)

          if (eventName === 'paragraphs_split') {
            totalParagraphs.value = parsed.total || 0
            rawParagraphs.value = parsed.paragraphs || []
          } else if (eventName === 'language_detected') {
            languageInfo.value = {
              sourceLang: parsed.sourceLang,
              sourceLangName: parsed.sourceLangName,
              confidence: parsed.confidence,
              targetLang: parsed.targetLang,
              targetLangName: parsed.targetLangName,
            }
            status.value = 'translating'
          } else if (eventName === 'paragraph_translated') {
            status.value = 'translating'
            const pair: ParagraphPair = {
              index: parsed.index,
              source: parsed.source,
              translation: parsed.translation,
            }
            translatedPairs.value = [...translatedPairs.value, pair]
            if (onPairReceived) {
              onPairReceived(pair)
            }
          } else if (eventName === 'done') {
            status.value = 'completed'
          } else if (eventName === 'paragraph_error') {
            errorMessage.value = parsed.message || 'Error translating paragraph'
          }
        }
      }

      if (status.value !== 'error') {
        status.value = 'completed'
      }
    } catch (err: any) {
      if (err?.name === 'AbortError') return
      status.value = 'error'
      errorMessage.value = err?.message || 'Unexpected streaming error'
    }
  }

  return {
    status,
    totalParagraphs,
    rawParagraphs,
    translatedPairs,
    languageInfo,
    errorMessage,
    startTranslationStream,
    stopStream,
    reset,
  }
}

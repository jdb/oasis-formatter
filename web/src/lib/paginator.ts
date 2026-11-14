export interface ParagraphPair {
  index: number
  source: string
  translation: string
}

export interface TypographyOptions {
  sourceFontSizePt: number
  translationFontSizePt: number
  lineHeightMultiplier: number
  marginTopPt: number
  marginBottomPt: number
  marginHorizontalPt: number
  pairGapPt: number
  interParagraphGapPt: number
  translationIndentPt: number
}

// Exact 7" Kindle Oasis (9th/10th Gen, 1264x1680 @ 300 PPI) physical & typographic dimensions
export const OASIS_WIDTH_MM = 107.02
export const OASIS_HEIGHT_MM = 142.24
export const OASIS_WIDTH_PT = 303.36  // (1264 / 300) * 72
export const OASIS_HEIGHT_PT = 403.20 // (1680 / 300) * 72

export const DEFAULT_TYPOGRAPHY: TypographyOptions = {
  sourceFontSizePt: 11.0,
  translationFontSizePt: 10.0,
  lineHeightMultiplier: 1.35,
  marginTopPt: 18,
  marginBottomPt: 24, // includes space for bottom page number
  marginHorizontalPt: 16,
  interParagraphGapPt: 4,
  pairGapPt: 11,
  translationIndentPt: 8,
}

export interface PageBlock {
  pairIndex: number
  kind: 'source' | 'translation'
  lines: string[]
  fontSizePt: number
  lineHeightPt: number
  yStartPt: number // top-down Y coordinate from top of page (0..403.20)
  heightPt: number
  indentPt: number
  isContinuation: boolean
}

export interface OasisPage {
  pageNumber: number
  widthPt: number
  heightPt: number
  blocks: PageBlock[]
}

export type MeasureTextFn = (text: string, fontSizePt: number, italic: boolean) => number

let sharedCanvasCtx: CanvasRenderingContext2D | null = null

export function createBrowserTextMeasurer(): MeasureTextFn {
  if (typeof document !== 'undefined') {
    const canvas = document.createElement('canvas')
    sharedCanvasCtx = canvas.getContext('2d')
  }
  return (text: string, fontSizePt: number, italic: boolean): number => {
    if (sharedCanvasCtx) {
      sharedCanvasCtx.font = `${italic ? 'italic ' : ''}400 ${fontSizePt}px Literata, Georgia, serif`
      return sharedCanvasCtx.measureText(text).width
    }
    return estimateTextWidthPt(text, fontSizePt, italic)
  }
}

export function estimateTextWidthPt(text: string, fontSizePt: number, italic: boolean): number {
  let total = 0
  const factor = italic ? 0.51 : 0.53
  for (let i = 0; i < text.length; i++) {
    const code = text.charCodeAt(i)
    if (code === 32) {
      total += fontSizePt * 0.27
    } else if ('ilI.,:;\'!|'.includes(text[i])) {
      total += fontSizePt * 0.28
    } else if ('mwMW@%'.includes(text[i])) {
      total += fontSizePt * 0.78
    } else if (code > 0x2e80) {
      // CJK wide glyphs
      total += fontSizePt * 0.98
    } else {
      total += fontSizePt * factor
    }
  }
  return total
}

export function wrapParagraphLines(
  text: string,
  maxWidthPt: number,
  fontSizePt: number,
  italic: boolean,
  measure: MeasureTextFn = estimateTextWidthPt,
): string[] {
  const normalized = text.replace(/\s+/g, ' ').trim()
  if (!normalized) return []

  // Tokenize so both space-delimited words and CJK characters wrap cleanly
  const tokens = normalized.match(/[\u2E80-\u9FFF\uF900-\uFAFF\u3040-\u30FF]|[^\s\u2E80-\u9FFF\uF900-\uFAFF\u3040-\u30FF]+/g) || [normalized]
  const lines: string[] = []
  let currentLine = ''

  for (const token of tokens) {
    const isCJK = /^[\u2E80-\u9FFF\uF900-\uFAFF\u3040-\u30FF]$/.test(token)
    const separator = currentLine === '' || isCJK ? '' : ' '
    const candidate = currentLine + separator + token

    if (currentLine === '' || measure(candidate, fontSizePt, italic) <= maxWidthPt) {
      currentLine = candidate
    } else {
      lines.push(currentLine)
      // If a single long token exceeds maxWidthPt, break it by character
      if (measure(token, fontSizePt, italic) > maxWidthPt) {
        let chunk = ''
        for (const ch of token) {
          if (chunk === '' || measure(chunk + ch, fontSizePt, italic) <= maxWidthPt) {
            chunk += ch
          } else {
            lines.push(chunk)
            chunk = ch
          }
        }
        currentLine = chunk
      } else {
        currentLine = token
      }
    }
  }

  if (currentLine) {
    lines.push(currentLine)
  }
  return lines
}

/**
 * Paginates an ordered array of (source, translation) paragraph pairs into exact
 * Kindle Oasis pages (303.36 pt × 403.20 pt).
 * Paragraph 1 of each pair is the source text; Paragraph 2 is its translation.
 */
export function paginateForKindleOasis(
  pairs: ParagraphPair[],
  opts: Partial<TypographyOptions> = {},
  measure: MeasureTextFn = estimateTextWidthPt,
): OasisPage[] {
  const typo: TypographyOptions = { ...DEFAULT_TYPOGRAPHY, ...opts }
  const contentWidthPt = OASIS_WIDTH_PT - typo.marginHorizontalPt * 2
  const maxContentY = OASIS_HEIGHT_PT - typo.marginBottomPt

  const srcLineHeightPt = +(typo.sourceFontSizePt * typo.lineHeightMultiplier).toFixed(2)
  const trLineHeightPt = +(typo.translationFontSizePt * typo.lineHeightMultiplier).toFixed(2)

  const pages: OasisPage[] = []
  let currentPage: OasisPage = {
    pageNumber: 1,
    widthPt: OASIS_WIDTH_PT,
    heightPt: OASIS_HEIGHT_PT,
    blocks: [],
  }
  let cursorY = typo.marginTopPt

  const startNewPage = () => {
    if (currentPage.blocks.length > 0) {
      pages.push(currentPage)
      currentPage = {
        pageNumber: pages.length + 1,
        widthPt: OASIS_WIDTH_PT,
        heightPt: OASIS_HEIGHT_PT,
        blocks: [],
      }
    }
    cursorY = typo.marginTopPt
  }

  const placeLinesWithOverflow = (
    pairIndex: number,
    kind: 'source' | 'translation',
    lines: string[],
    fontSizePt: number,
    lineHeightPt: number,
    indentPt: number,
  ) => {
    let remaining = [...lines]
    let isContinuation = false

    while (remaining.length > 0) {
      const availablePt = maxContentY - cursorY
      const maxLinesHere = Math.floor(availablePt / lineHeightPt)

      if (maxLinesHere <= 0) {
        startNewPage()
        continue
      }

      const slice = remaining.slice(0, maxLinesHere)
      remaining = remaining.slice(maxLinesHere)
      const blockHeightPt = +(slice.length * lineHeightPt).toFixed(2)

      currentPage.blocks.push({
        pairIndex,
        kind,
        lines: slice,
        fontSizePt,
        lineHeightPt,
        yStartPt: +cursorY.toFixed(2),
        heightPt: blockHeightPt,
        indentPt,
        isContinuation,
      })

      cursorY += blockHeightPt
      isContinuation = true

      if (remaining.length > 0) {
        startNewPage()
      }
    }
  }

  for (const pair of pairs) {
    const srcLines = wrapParagraphLines(
      pair.source,
      contentWidthPt,
      typo.sourceFontSizePt,
      false,
      measure,
    )
    const trLines = wrapParagraphLines(
      pair.translation,
      contentWidthPt - typo.translationIndentPt,
      typo.translationFontSizePt,
      true,
      measure,
    )

    if (srcLines.length === 0 && trLines.length === 0) continue

    const srcHeightPt = srcLines.length * srcLineHeightPt
    const trHeightPt = trLines.length * trLineHeightPt
    const pairTotalHeightPt = srcHeightPt + typo.interParagraphGapPt + trHeightPt
    const singlePageCapacityPt = maxContentY - typo.marginTopPt

    // Keep source + translation together on the same page whenever the pair fits on a fresh page
    // and would otherwise overflow the current page.
    if (
      currentPage.blocks.length > 0 &&
      cursorY + pairTotalHeightPt > maxContentY &&
      pairTotalHeightPt <= singlePageCapacityPt
    ) {
      startNewPage()
    } else if (
      currentPage.blocks.length > 0 &&
      cursorY + Math.min(srcHeightPt, srcLineHeightPt * 2) > maxContentY
    ) {
      // Avoid leaving a single orphan line of the source paragraph at the bottom of a page
      startNewPage()
    }

    // 1. Place Source Paragraph (First Paragraph)
    placeLinesWithOverflow(
      pair.index,
      'source',
      srcLines,
      typo.sourceFontSizePt,
      srcLineHeightPt,
      0,
    )

    cursorY += typo.interParagraphGapPt

    // 2. Place Translation Paragraph (Second Paragraph)
    placeLinesWithOverflow(
      pair.index,
      'translation',
      trLines,
      typo.translationFontSizePt,
      trLineHeightPt,
      typo.translationIndentPt,
    )

    cursorY += typo.pairGapPt
  }

  if (currentPage.blocks.length > 0) {
    pages.push(currentPage)
  }

  return pages
}

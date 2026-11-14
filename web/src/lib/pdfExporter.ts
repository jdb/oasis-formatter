import { PDFDocument, rgb, StandardFonts } from 'pdf-lib'
import fontkit from '@pdf-lib/fontkit'
import {
  DEFAULT_TYPOGRAPHY,
  OASIS_HEIGHT_PT,
  OASIS_WIDTH_PT,
  type OasisPage,
  type TypographyOptions,
} from './paginator'

export interface PDFExportMetadata {
  title?: string
  sourceLangName?: string
  targetLangName?: string
}

let cachedRegularBytes: ArrayBuffer | null = null
let cachedItalicBytes: ArrayBuffer | null = null

async function loadSelfHostedFontBytes(url: string): Promise<ArrayBuffer | null> {
  try {
    const resp = await fetch(url)
    if (!resp.ok) return null
    return await resp.arrayBuffer()
  } catch {
    return null
  }
}

/**
 * Generates a 1:1 Kindle Oasis vector PDF (303.36 pt × 403.20 pt / 107.02 mm × 142.24 mm)
 * where every page and line matches the live Kindle Oasis paginator preview.
 */
export async function generateKindleOasisPDF(
  pages: OasisPage[],
  typo: Partial<TypographyOptions> = {},
  meta: PDFExportMetadata = {},
): Promise<Uint8Array> {
  const opts: TypographyOptions = { ...DEFAULT_TYPOGRAPHY, ...typo }
  const pdfDoc = await PDFDocument.create()
  pdfDoc.registerFontkit(fontkit)

  pdfDoc.setTitle(meta.title || 'Oasis Formatter — Bilingual Kindle Oasis Edition')
  pdfDoc.setCreator('Oasis Formatter (Kindle Oasis 107.02 × 142.24 mm)')
  pdfDoc.setProducer('Oasis Formatter')

  if (!cachedRegularBytes) {
    cachedRegularBytes = await loadSelfHostedFontBytes('/fonts/Literata-Regular.ttf')
  }
  if (!cachedItalicBytes) {
    cachedItalicBytes = await loadSelfHostedFontBytes('/fonts/Literata-Italic.ttf')
  }

  const regularFont = cachedRegularBytes
    ? await pdfDoc.embedFont(cachedRegularBytes, { subset: true })
    : await pdfDoc.embedFont(StandardFonts.TimesRoman)

  const italicFont = cachedItalicBytes
    ? await pdfDoc.embedFont(cachedItalicBytes, { subset: true })
    : await pdfDoc.embedFont(StandardFonts.TimesRomanItalic)

  for (const oasisPage of pages) {
    // Exact Kindle Oasis page size in points: [303.36, 403.20]
    const page = pdfDoc.addPage([OASIS_WIDTH_PT, OASIS_HEIGHT_PT])

    for (const block of oasisPage.blocks) {
      const isTranslation = block.kind === 'translation'
      const font = isTranslation ? italicFont : regularFont
      const textColor = isTranslation ? rgb(0.18, 0.18, 0.18) : rgb(0, 0, 0)
      const xStart = opts.marginHorizontalPt + block.indentPt

      // Draw subtle left border for the translated second paragraph
      if (isTranslation && block.lines.length > 0) {
        const barX = opts.marginHorizontalPt + 2.5
        const topPdfY = OASIS_HEIGHT_PT - block.yStartPt - 2
        const bottomPdfY = OASIS_HEIGHT_PT - (block.yStartPt + block.heightPt) + 2
        page.drawLine({
          start: { x: barX, y: topPdfY },
          end: { x: barX, y: bottomPdfY },
          thickness: 1.2,
          color: rgb(0.42, 0.42, 0.42),
        })
      }

      block.lines.forEach((lineText, lineIdx) => {
        // Convert top-down Y coordinate to PDF bottom-up baseline Y coordinate
        const baselineFromTop =
          block.yStartPt + lineIdx * block.lineHeightPt + block.fontSizePt * 0.84
        const pdfY = OASIS_HEIGHT_PT - baselineFromTop

        page.drawText(lineText, {
          x: xStart,
          y: pdfY,
          size: block.fontSizePt,
          font,
          color: textColor,
        })
      })
    }

    // Minimalist footer page number centered at bottom
    const footerText = `${oasisPage.pageNumber} / ${pages.length}`
    const footerSize = 7.5
    const footerWidth = regularFont.widthOfTextAtSize(footerText, footerSize)
    page.drawText(footerText, {
      x: (OASIS_WIDTH_PT - footerWidth) / 2,
      y: 9,
      size: footerSize,
      font: regularFont,
      color: rgb(0.45, 0.45, 0.45),
    })
  }

  return await pdfDoc.save()
}

export function triggerPDFDownload(pdfBytes: Uint8Array, filename = 'kindle-oasis-bilingual.pdf') {
  const blob = new Blob([pdfBytes], { type: 'application/pdf' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  setTimeout(() => URL.revokeObjectURL(url), 3000)
}

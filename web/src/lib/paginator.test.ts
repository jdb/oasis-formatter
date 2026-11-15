import { describe, expect, it } from 'vitest'
import { PDFDocument } from 'pdf-lib'
import {
  OASIS_HEIGHT_PT,
  OASIS_WIDTH_PT,
  paginateForKindleOasis,
} from './paginator'
import { generateKindleOasisPDF } from './pdfExporter'

describe('paginateForKindleOasis', () => {
  it('creates exact Kindle Oasis pages (303.36pt x 403.20pt) with source paragraph first and translation second', () => {
    const pairs = [
      {
        index: 0,
        source: 'Longtemps, je me suis couché de bonne heure.',
        translation: 'For a long time I used to go to bed early.',
      },
      {
        index: 1,
        source: 'Parfois, à peine ma bougie éteinte, mes yeux se fermaient si vite que je n’avais pas le temps de me dire : « Je m’endors. »',
        translation: 'Sometimes, when I had put out my candle, my eyes would close so quickly that I had not even time to say to myself: "I am falling asleep."',
      },
    ]

    const pages = paginateForKindleOasis(pairs)
    expect(pages.length).toBeGreaterThanOrEqual(1)
    expect(pages[0].widthPt).toBeCloseTo(303.36, 2)
    expect(pages[0].heightPt).toBeCloseTo(403.20, 2)

    const blocks = pages[0].blocks
    expect(blocks[0].kind).toBe('source')
    expect(blocks[0].pairIndex).toBe(0)
    expect(blocks[1].kind).toBe('translation')
    expect(blocks[1].pairIndex).toBe(0)
    expect(blocks[1].yStartPt).toBeGreaterThan(blocks[0].yStartPt)
  })

  it('spawns additional Kindle Oasis pages when paragraph pairs overflow a single page', () => {
    const pairs = Array.from({ length: 14 }, (_, i) => ({
      index: i,
      source: `Paragraphe numéro ${i + 1} avec suffisamment de texte en français pour occuper plusieurs lignes sur l'écran liseuse Kindle Oasis de sept pouces.`,
      translation: `Paragraph number ${i + 1} with enough translated text in English to occupy multiple lines on the seven-inch Kindle Oasis e-reader display.`,
    }))

    const pages = paginateForKindleOasis(pairs)
    expect(pages.length).toBeGreaterThan(1)
    for (const p of pages) {
      expect(p.widthPt).toBe(OASIS_WIDTH_PT)
      expect(p.heightPt).toBe(OASIS_HEIGHT_PT)
      for (const b of p.blocks) {
        expect(b.yStartPt + b.heightPt).toBeLessThanOrEqual(OASIS_HEIGHT_PT)
      }
    }
  })

  it('generates a valid 1:1 Kindle Oasis vector PDF matching the paginated page count and dimensions', async () => {
    const pairs = [
      {
        index: 0,
        source: 'Bonjour tout le monde, voici le premier paragraphe.',
        translation: 'Hello everyone, here is the first paragraph.',
      },
    ]
    const pages = paginateForKindleOasis(pairs)
    const pdfBytes = await generateKindleOasisPDF(pages, {}, { title: 'Test Oasis PDF' })
    expect(pdfBytes.byteLength).toBeGreaterThan(500)

    const parsedDoc = await PDFDocument.load(pdfBytes)
    expect(parsedDoc.getPageCount()).toBe(1)
    const { width, height } = parsedDoc.getPage(0).getSize()
    expect(width).toBeCloseTo(OASIS_WIDTH_PT, 2)
    expect(height).toBeCloseTo(OASIS_HEIGHT_PT, 2)
  })

  it('supports format-only mode (no translation) without adding empty translation blocks', () => {
    const pairs = [
      { index: 0, source: 'First paragraph without translation.', translation: '' },
      { index: 1, source: 'Second paragraph without translation.', translation: '' },
    ]
    const pages = paginateForKindleOasis(pairs)
    expect(pages.length).toBe(1)
    expect(pages[0].blocks.length).toBe(2)
    expect(pages[0].blocks.every((b) => b.kind === 'source')).toBe(true)
  })
})

<script setup lang="ts">
import { ref } from 'vue'

defineProps<{
  isBusy: boolean
}>()

const emit = defineEmits<{
  (e: 'submit-text', text: string): void
}>()

const inputText = ref('')
const isDragging = ref(false)

const samples = [
  {
    label: 'French — Marcel Proust (Du côté de chez Swann)',
    text: `Longtemps, je me suis couché de bonne heure. Parfois, à peine ma bougie éteinte, mes yeux se fermaient si vite que je n’avais pas le temps de me dire : « Je m’endors. » Et, une demi-heure après, la pensée qu’il était temps de chercher le sommeil m’éveillait ; je voulais poser le volume que je croyais avoir encore dans les mains et souffler ma lumière ; je n’avais pas cessé en dormant de faire des réflexions sur ce que je venais de lire, mais ces réflexions avaient pris un tour un peu particulier ; il me semblait que j’étais moi-même ce dont parlait l’ouvrage : une église, un quatuor, la rivalité de François Ier et de Charles-Quint.

Cette croyance survivait pendant quelques secondes à mon réveil ; elle ne choquait pas ma raison, mais pesait comme des écailles sur mes yeux et les empêchait de se rendre compte que le bougeoir n’était plus allumé. Puis elle commençait à me devenir inintelligible, comme après la métempsycose les pensées d’une existence antérieure ; le sujet du livre se détachait de moi, j’étais libre de m’y appliquer ou non ; aussitôt je recouvrais la vue et j’étais bien étonné de trouver autour de moi une obscurité, douce et reposante pour mes yeux, mais peut-être plus encore pour mon esprit, à qui elle apparaissait comme une chose sans cause, incompréhensible, comme une chose vraiment obscure.

Je me demandais quelle heure il pouvait être ; j’entendais le sifflement des trains qui, plus ou moins éloigné, comme le chant d’un oiseau dans une forêt, relevant les distances, me décrivait l’étendue de la campagne déserte où le voyageur se hâte vers la station prochaine ; et le petit chemin qu’il suit va être gravé dans son souvenir par l’excitation qu’il doit à des lieux nouveaux, à des actes inaccoutumés, à la causerie récente et aux adieux sous la lampe étrangère qui le suivent encore dans le silence de la nuit, à la douceur prochaine du retour.`,
  },
  {
    label: 'German — Franz Kafka (Der Process)',
    text: `Jemand musste Josef K. verleumdet haben, denn ohne dass er etwas Böses getan hätte, wurde er eines Morgens verhaftet. Die Köchin der Frau Grubach, seiner Zimmervermieterin, die ihm jeden Tag gegen acht Uhr früh das Frühstück brachte, kam diesmal nicht. Das war noch niemals geschehen. K. wartete noch ein Weilchen, sah von seinem Kopfkissen aus die alte Frau, die ihm gegenüber wohnte und die ihn mit einer an ihr ganz ungewöhnlichen Neugierde beobachtete, aber dann, gleichzeitig befremdet und hungrig, läutete er.

Sofort klopfte es und ein Mann, den er in dieser Wohnung noch niemals gesehen hatte, trat ein. Er war schlank und doch fest gebaut, er trug ein anliegendes schwarzes Kleid, das, ähnlich den Reiseanzügen, mit verschiedenen Falten, Taschen, Schnallen, Knöpfen und einem Gürtel versehen war und infolgedessen, ohne dass man sich darüber klar wurde, wozu es dienen sollte, besonders praktisch erschien.

»Wer sind Sie?« fragte K. und saß gleich halb aufrecht im Bett. Der Mann aber ging über die Frage hinweg, als müsse man seine Erscheinung hinnehmen, und sagte bloß seinerseits: »Sie haben geläutet?«`,
  },
  {
    label: 'Spanish — Miguel de Cervantes (Don Quijote)',
    text: `En un lugar de la Mancha, de cuyo nombre no quiero acordarme, no ha mucho tiempo que vivía un hidalgo de los de lanza en astillero, adarga antigua, rocín flaco y galgo corredor. Una olla de algo más vaca que carnero, salpicón las más noches, duelos y quebrantos los sábados, lantejas los viernes, algún palomino de añadidura los domingos, consumían las tres partes de su hacienda.

El resto della concluían sayo de velarte, calzas de velludo para las fiestas, con sus pantuflos de lo mesmo, y los días de entresemana se honraba con su vellorí de lo más fino. Tenía en su casa una ama que pasaba de los cuarenta, y una sobrina que no llegaba a los veinte, y un mozo de campo y plaza, que así ensillaba el rocín como tomaba la podadera.

Frisaba la edad de nuestro hidalgo con los cincuenta años; era de complexión recia, seco de carnes, enjuto de rostro, gran madrugador y amigo de la caza. Quieren decir que tenía el sobrenombre de Quijada, o Quesada, que en esto hay alguna diferencia en los autores que deste caso escriben.`,
  },
]

function onDragOver(e: DragEvent) {
  e.preventDefault()
  isDragging.value = true
}

function onDragLeave() {
  isDragging.value = false
}

async function onDrop(e: DragEvent) {
  e.preventDefault()
  isDragging.value = false

  if (e.dataTransfer?.files && e.dataTransfer.files.length > 0) {
    const file = e.dataTransfer.files[0]
    const text = await file.text()
    if (text.trim()) {
      inputText.value = text
      emit('submit-text', text)
    }
    return
  }

  const droppedText = e.dataTransfer?.getData('text/plain')
  if (droppedText && droppedText.trim()) {
    inputText.value = droppedText
    emit('submit-text', droppedText)
  }
}

async function onFileSelected(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return
  const text = await input.files[0].text()
  if (text.trim()) {
    inputText.value = text
    emit('submit-text', text)
  }
}

function onPasteText(e: ClipboardEvent) {
  const pasted = e.clipboardData?.getData('text/plain')
  if (pasted && pasted.trim() && !inputText.value.trim()) {
    e.preventDefault()
    inputText.value = pasted
    emit('submit-text', pasted)
  }
}

function submitCurrentText() {
  if (!inputText.value.trim()) return
  emit('submit-text', inputText.value)
}

function loadSample(text: string) {
  inputText.value = text
  emit('submit-text', text)
}
</script>

<template>
  <section
    class="drop-zone"
    :class="{ dragging: isDragging }"
    @dragover="onDragOver"
    @dragleave="onDragLeave"
    @drop="onDrop"
  >
    <div class="drop-header">
      <div>
        <h2 class="drop-title">Drop text or a file to start translating</h2>
        <p class="drop-subtitle">
          Drop a <code>.txt</code> / <code>.md</code> file or paste text below — source language is auto-detected and Kindle Oasis pages materialize live as paragraphs translate.
        </p>
      </div>
      <label class="file-upload-btn">
        <span>Choose .txt / .md file</span>
        <input type="file" accept=".txt,.md,text/plain" hidden @change="onFileSelected" />
      </label>
    </div>

    <textarea
      v-model="inputText"
      class="text-input"
      rows="5"
      placeholder="Drop a file anywhere here, paste your text (auto-starts on paste), or pick a sample below…"
      @paste="onPasteText"
    />

    <div class="drop-footer">
      <div class="samples-row">
        <span class="samples-label">Try sample:</span>
        <button
          v-for="sample in samples"
          :key="sample.label"
          type="button"
          class="sample-chip"
          @click="loadSample(sample.text)"
        >
          {{ sample.label }}
        </button>
      </div>

      <button
        type="button"
        class="translate-btn"
        :disabled="!inputText.trim() || isBusy"
        @click="submitCurrentText"
      >
        {{ isBusy ? 'Translating…' : 'Translate & Format for Kindle Oasis' }}
      </button>
    </div>
  </section>
</template>

<style scoped>
.drop-zone {
  background: #ffffff;
  border: 2px dashed #cfc9bc;
  border-radius: 14px;
  padding: 1.25rem;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.drop-zone.dragging {
  border-color: #1c1b18;
  background: #f5f2ea;
}

.drop-header {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  margin-bottom: 0.85rem;
}

.drop-title {
  margin: 0;
  font-size: 1.05rem;
  font-weight: 700;
  color: #1c1b18;
}

.drop-subtitle {
  margin: 0.25rem 0 0;
  font-size: 0.86rem;
  color: #625d55;
}

.file-upload-btn {
  display: inline-flex;
  align-items: center;
  padding: 0.45rem 0.85rem;
  background: #f4f1ea;
  border: 1px solid #d5d0c5;
  border-radius: 8px;
  font-size: 0.82rem;
  font-weight: 600;
  color: #2c2a25;
  cursor: pointer;
}

.file-upload-btn:hover {
  background: #eae5da;
}

.text-input {
  width: 100%;
  box-sizing: border-box;
  padding: 0.75rem 0.9rem;
  border: 1px solid #dedad0;
  border-radius: 10px;
  background: #fcfbf8;
  font-family: 'Literata', Georgia, serif;
  font-size: 0.92rem;
  line-height: 1.45;
  color: #1c1b18;
  resize: vertical;
}

.text-input:focus {
  outline: none;
  border-color: #8a8377;
}

.drop-footer {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 0.75rem;
  margin-top: 0.85rem;
}

.samples-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.45rem;
}

.samples-label {
  font-size: 0.78rem;
  font-weight: 600;
  color: #6e695f;
}

.sample-chip {
  padding: 0.3rem 0.65rem;
  font-size: 0.78rem;
  border: 1px solid #dcd7cc;
  border-radius: 999px;
  background: #faf8f3;
  color: #36332d;
  cursor: pointer;
}

.sample-chip:hover {
  background: #ece8de;
}

.translate-btn {
  padding: 0.55rem 1.1rem;
  border-radius: 8px;
  border: none;
  background: #1c1b18;
  color: #faf9f5;
  font-size: 0.88rem;
  font-weight: 600;
  cursor: pointer;
}

.translate-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>

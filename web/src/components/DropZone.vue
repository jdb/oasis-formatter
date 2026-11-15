<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', text: string): void
}>()

const isDragging = ref(false)

const samples = [
  {
    label: 'French (Proust)',
    text: `Longtemps, je me suis couché de bonne heure. Parfois, à peine ma bougie éteinte, mes yeux se fermaient si vite que je n’avais pas le temps de me dire : « Je m’endors. » Et, une demi-heure après, la pensée qu’il était temps de chercher le sommeil m’éveillait ; je voulais poser le volume que je croyais avoir encore dans les mains et souffler ma lumière ; je n’avais pas cessé en dormant de faire des réflexions sur ce que je venais de lire, mais ces réflexions avaient pris un tour un peu particulier ; il me semblait que j’étais moi-même ce dont parlait l’ouvrage : une église, un quatuor, la rivalité de François Ier et de Charles-Quint.

Cette croyance survivait pendant quelques secondes à mon réveil ; elle ne choquait pas ma raison, mais pesait comme des écailles sur mes yeux et les empêchait de se rendre compte que le bougeoir n’était plus allumé. Puis elle commençait à me devenir inintelligible, comme après la métempsycose les pensées d’une existence antérieure ; le sujet du livre se détachait de moi, j’étais libre de m’y appliquer ou non ; aussitôt je recouvrais la vue et j’étais bien étonné de trouver autour de moi une obscurité, douce et reposante pour mes yeux, mais peut-être plus encore pour mon esprit, à qui elle apparaissait comme une chose sans cause, incompréhensible, comme une chose vraiment obscure.`,
  },
  {
    label: 'German (Kafka)',
    text: `Jemand musste Josef K. verleumdet haben, denn ohne dass er etwas Böses getan hätte, wurde er eines Morgens verhaftet. Die Köchin der Frau Grubach, seiner Zimmervermieterin, die ihm jeden Tag gegen acht Uhr früh das Frühstück brachte, kam diesmal nicht. Das war noch niemals geschehen. K. wartete noch ein Weilchen, sah von seinem Kopfkissen aus die alte Frau, die ihm gegenüber wohnte und die ihn mit einer an ihr ganz ungewöhnlichen Neugierde beobachtete, aber dann, gleichzeitig befremdet und hungrig, läutete er.

Sofort klopfte es und ein Mann, den er in dieser Wohnung noch niemals gesehen hatte, trat ein. Er war schlank und doch fest gebaut, er trug ein anliegendes schwarzes Kleid, das, ähnlich den Reiseanzügen, mit verschiedenen Falten, Taschen, Schnallen, Knöpfen und einem Gürtel versehen war und infolgedessen, ohne dass man sich darüber klar wurde, wozu es dienen sollte, besonders praktisch erschien.`,
  },
  {
    label: 'Spanish (Cervantes)',
    text: `En un lugar de la Mancha, de cuyo nombre no quiero acordarme, no ha mucho tiempo que vivía un hidalgo de los de lanza en astillero, adarga antigua, rocín flaco y galgo corredor. Una olla de algo más vaca que carnero, salpicón las más noches, duelos y quebrantos los sábados, lantejas los viernes, algún palomino de añadidura los domingos, consumían las tres partes de su hacienda.

El resto della concluían sayo de velarte, calzas de velludo para las fiestas, con sus pantuflos de lo mesmo, y los días de entresemana se honraba con su vellorí de lo más fino. Tenía en su casa una ama que pasaba de los cuarenta, y una sobrina que no llegaba a los veinte, y un mozo de campo y plaza, que así ensillaba el rocín como tomaba la podadera.`,
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
    const text = await e.dataTransfer.files[0].text()
    if (text.trim()) {
      emit('update:modelValue', text)
    }
    return
  }

  const droppedText = e.dataTransfer?.getData('text/plain')
  if (droppedText && droppedText.trim()) {
    emit('update:modelValue', droppedText)
  }
}

async function onFileSelected(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return
  const text = await input.files[0].text()
  if (text.trim()) {
    emit('update:modelValue', text)
  }
}
</script>

<template>
  <section
    class="step-card"
    :class="{ 'is-dragging': isDragging }"
    @dragover="onDragOver"
    @dragleave="onDragLeave"
    @drop="onDrop"
  >
    <div class="step-header">
      <h2 class="step-title">1. Drop the document</h2>
      <label class="btn-subtle">
        <span>Choose .txt / .md</span>
        <input type="file" accept=".txt,.md,text/plain" hidden @change="onFileSelected" />
      </label>
    </div>

    <textarea
      :value="modelValue"
      class="text-input"
      rows="4"
      placeholder="Drop a .txt or .md file here, or paste your text…"
      @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
    />

    <div class="samples-row">
      <span class="muted-label">Samples:</span>
      <button
        v-for="sample in samples"
        :key="sample.label"
        type="button"
        class="btn-link"
        @click="emit('update:modelValue', sample.text)"
      >
        {{ sample.label }}
      </button>
    </div>
  </section>
</template>

<style scoped>
.step-card.is-dragging {
  border-color: #1c1b18;
  background: #faf8f4;
}

.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  margin-bottom: 0.75rem;
}

.text-input {
  width: 100%;
  box-sizing: border-box;
  padding: 0.7rem 0.85rem;
  border: 1px solid #dedad0;
  border-radius: 6px;
  background: #fcfbf9;
  font-family: inherit;
  font-size: 0.92rem;
  line-height: 1.5;
  color: #1c1b18;
  resize: vertical;
}

.text-input:focus {
  outline: none;
  border-color: #8c867a;
}

.samples-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
  margin-top: 0.65rem;
}

.muted-label {
  font-size: 0.84rem;
  color: #6b665e;
}

.btn-link {
  padding: 0;
  border: none;
  background: none;
  font-family: inherit;
  font-size: 0.84rem;
  color: #1c1b18;
  text-decoration: underline;
  text-underline-offset: 3px;
  cursor: pointer;
}

.btn-link:hover {
  color: #5b564e;
}
</style>

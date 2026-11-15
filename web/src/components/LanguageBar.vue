<script setup lang="ts">
export interface LanguageOption {
  code: string
  name: string
}

defineProps<{
  languages: LanguageOption[]
  targetLang: string
  sourceFontSizePt: number
}>()

const emit = defineEmits<{
  (e: 'update:targetLang', code: string): void
  (e: 'update:sourceFontSizePt', size: number): void
}>()
</script>

<template>
  <section class="step-card">
    <div class="step-header">
      <h2 class="step-title">2. Options</h2>
    </div>

    <div class="options-row">
      <label class="field">
        <span class="field-label">Destination language</span>
        <select
          class="select-input"
          :value="targetLang"
          @change="emit('update:targetLang', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">No translation</option>
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
    </div>
  </section>
</template>

<style scoped>
.step-header {
  margin-bottom: 0.85rem;
}

.options-row {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 1.5rem;
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
</style>

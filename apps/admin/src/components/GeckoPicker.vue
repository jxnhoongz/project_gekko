<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { X, ChevronDown } from 'lucide-vue-next';
import { useGeckos } from '@/composables/useGeckos';
import type { Sex } from '@/types/gecko';

const props = defineProps<{
  modelValue: number | null;
  speciesId: number | null;
  sex: Sex;           // 'M' | 'F' (sire / dam). 'U' accepted for completeness.
  excludeId?: number; // don't allow self-selection
  placeholder?: string;
}>();

const emit = defineEmits<{ (e: 'update:modelValue', v: number | null): void }>();

const { data: geckosData } = useGeckos();
const allGeckos = computed(() => geckosData.value?.geckos ?? []);

const open = ref(false);
const query = ref('');
const inputRef = ref<HTMLInputElement | null>(null);

const candidates = computed(() => {
  if (props.speciesId === null) return [];
  return allGeckos.value.filter(
    (g) =>
      g.species_id === props.speciesId &&
      g.sex === props.sex &&
      g.id !== props.excludeId,
  );
});

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return candidates.value;
  return candidates.value.filter(
    (g) =>
      g.code.toLowerCase().includes(q) ||
      (g.name ?? '').toLowerCase().includes(q),
  );
});

const selected = computed(() =>
  allGeckos.value.find((g) => g.id === props.modelValue) ?? null,
);

const display = computed(() => {
  if (!selected.value) return '';
  return `${selected.value.code}${selected.value.name ? ' · ' + selected.value.name : ''}`;
});

function pick(id: number) {
  emit('update:modelValue', id);
  open.value = false;
  query.value = '';
}

function clear(e: Event) {
  e.stopPropagation();
  emit('update:modelValue', null);
  query.value = '';
}

function openPicker() {
  if (props.speciesId === null) return;
  open.value = true;
  setTimeout(() => inputRef.value?.focus(), 0);
}

// Close on outside click
const wrapperRef = ref<HTMLDivElement | null>(null);
function onDocClick(e: MouseEvent) {
  if (!wrapperRef.value) return;
  if (!wrapperRef.value.contains(e.target as Node)) open.value = false;
}

watch(open, (v) => {
  if (v) document.addEventListener('mousedown', onDocClick);
  else document.removeEventListener('mousedown', onDocClick);
});
</script>

<template>
  <div ref="wrapperRef" class="relative">
    <!-- Closed state: shows selected pill or placeholder -->
    <button
      v-if="!open"
      type="button"
      class="w-full h-9 rounded-md border border-brand-cream-300 bg-white px-3 text-left text-sm flex items-center gap-2 transition-colors disabled:bg-brand-cream-100 disabled:text-brand-dark-400 disabled:cursor-not-allowed"
      :disabled="speciesId === null"
      @click="openPicker"
    >
      <span v-if="speciesId === null" class="text-brand-dark-500">Pick species first</span>
      <span v-else-if="selected" class="flex-1 truncate text-brand-dark-950">{{ display }}</span>
      <span v-else class="flex-1 truncate text-brand-dark-500">{{ placeholder ?? 'Select…' }}</span>
      <button
        v-if="selected"
        type="button"
        class="size-5 rounded hover:bg-brand-cream-200 flex items-center justify-center"
        aria-label="Clear"
        @click="clear"
      >
        <X class="size-3" />
      </button>
      <ChevronDown v-else class="size-4 text-brand-dark-500 shrink-0" />
    </button>

    <!-- Open state: search input + dropdown -->
    <div
      v-else
      class="absolute inset-x-0 top-0 z-30 rounded-md border border-brand-cream-300 bg-white shadow-lg"
    >
      <input
        ref="inputRef"
        v-model="query"
        type="text"
        :placeholder="placeholder ?? 'Type to search…'"
        class="w-full h-9 px-3 text-sm border-b border-brand-cream-200 outline-none"
        @keydown.esc="open = false"
      />
      <ul
        v-if="filtered.length"
        class="max-h-60 overflow-y-auto py-1"
        role="listbox"
      >
        <li
          v-for="g in filtered"
          :key="g.id"
          class="px-3 py-1.5 text-sm cursor-pointer hover:bg-brand-cream-100 flex items-center gap-2"
          @click="pick(g.id)"
        >
          <span class="font-mono text-brand-dark-700">{{ g.code }}</span>
          <span v-if="g.name" class="text-brand-dark-950">· {{ g.name }}</span>
        </li>
      </ul>
      <div v-else class="px-3 py-4 text-xs text-brand-dark-500 text-center">
        No matching geckos.
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, onMounted, onUnmounted } from 'vue';
import {
  DialogRoot,
  DialogPortal,
  DialogOverlay,
  DialogContent,
} from 'reka-ui';
import { X, ChevronLeft, ChevronRight, AlertTriangle } from 'lucide-vue-next';
import type { Trait, Species } from '@/types/gecko';
import type { InheritanceType } from '@/types/morph';
import { INHERITANCE_TYPE_LABEL } from '@/types/morph';

const props = defineProps<{
  open: boolean;
  traits: Trait[];
  modelValue: Trait | null;
  speciesById: Record<number, Species>;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  'update:modelValue': [value: Trait | null];
}>();

const currentIndex = computed(() =>
  props.modelValue ? props.traits.findIndex((t) => t.id === props.modelValue!.id) : -1,
);

const hasPrev = computed(() => currentIndex.value > 0);
const hasNext = computed(() => currentIndex.value < props.traits.length - 1);

function prev() {
  if (hasPrev.value) emit('update:modelValue', props.traits[currentIndex.value - 1]);
}
function next() {
  if (hasNext.value) emit('update:modelValue', props.traits[currentIndex.value + 1]);
}

function onKey(e: KeyboardEvent) {
  if (!props.open) return;
  if (e.key === 'ArrowLeft') prev();
  if (e.key === 'ArrowRight') next();
}

onMounted(() => window.addEventListener('keydown', onKey));
onUnmounted(() => window.removeEventListener('keydown', onKey));

const INHERITANCE_BADGE: Record<InheritanceType, { label: string; class: string }> = {
  RECESSIVE:   { label: 'Recessive',   class: 'bg-sky-100 text-sky-800 border-sky-200' },
  CO_DOMINANT: { label: 'Co-Dom',      class: 'bg-amber-100 text-amber-800 border-amber-200' },
  DOMINANT:    { label: 'Dominant',    class: 'bg-red-100 text-red-700 border-red-200' },
  POLYGENIC:   { label: 'Polygenic',   class: 'bg-brand-cream-200 text-brand-dark-700 border-brand-cream-300' },
};

const HEALTH_WARNING_TRAITS = ['Enigma', 'Lemon Frost'];
function hasHealthWarning(t: Trait) {
  return HEALTH_WARNING_TRAITS.some((n) => t.trait_name.includes(n));
}

const t = computed(() => props.modelValue);
</script>

<template>
  <DialogRoot :open="open" @update:open="emit('update:open', $event)">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-40 bg-brand-dark-950/40 backdrop-blur-sm" />
      <DialogContent
        class="fixed right-0 top-0 z-50 h-full w-full max-w-md bg-brand-cream-50 shadow-xl flex flex-col outline-none"
        @escape-key-down="emit('update:open', false)"
      >
        <!-- Header -->
        <div class="flex items-center justify-between border-b border-brand-cream-300 px-6 py-4 shrink-0">
          <div class="min-w-0">
            <p class="text-xs text-brand-dark-600 font-mono uppercase tracking-wide">
              {{ t ? (speciesById[t.species_id]?.code ?? `sp#${t.species_id}`) : '' }}
            </p>
            <h2 class="text-xl font-display text-brand-dark-950 truncate">
              {{ t?.trait_name ?? '' }}
            </h2>
          </div>
          <button
            aria-label="Close"
            class="ml-4 shrink-0 p-1 text-brand-dark-500 hover:text-brand-dark-950 rounded"
            @click="emit('update:open', false)"
          >
            <X class="w-5 h-5" />
          </button>
        </div>

        <!-- Scrollable body -->
        <div class="flex-1 overflow-y-auto px-6 py-5 space-y-5">
          <!-- Health warning -->
          <div
            v-if="t && hasHealthWarning(t)"
            class="flex gap-2 rounded-lg bg-amber-50 border border-amber-200 px-4 py-3 text-sm text-amber-800"
          >
            <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5 text-amber-500" />
            <span>{{ t.notes }}</span>
          </div>

          <!-- Photo -->
          <div
            v-if="t?.example_photo_url"
            class="rounded-xl overflow-hidden border border-brand-cream-300 bg-brand-cream-100"
          >
            <img
              :src="t.example_photo_url"
              :alt="t.trait_name"
              class="w-full object-cover max-h-64"
            />
          </div>
          <div
            v-else
            class="rounded-xl border border-dashed border-brand-cream-300 bg-brand-cream-100 flex items-center justify-center h-36 text-brand-dark-500 text-sm"
          >
            No photo
          </div>

          <!-- Detail grid -->
          <div class="grid grid-cols-2 gap-4 text-sm">
            <div>
              <p class="text-xs text-brand-dark-600 mb-1">Code</p>
              <p class="font-mono text-brand-dark-950">{{ t?.trait_code || '—' }}</p>
            </div>
            <div>
              <p class="text-xs text-brand-dark-600 mb-1">Species</p>
              <p class="text-brand-dark-950">
                {{ t ? (speciesById[t.species_id]?.common_name ?? `#${t.species_id}`) : '—' }}
              </p>
            </div>
            <div>
              <p class="text-xs text-brand-dark-600 mb-1">Inheritance</p>
              <span
                v-if="t?.inheritance_type"
                class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium"
                :class="INHERITANCE_BADGE[t.inheritance_type]?.class ?? ''"
              >
                {{ INHERITANCE_BADGE[t.inheritance_type]?.label ?? t.inheritance_type }}
              </span>
            </div>
            <div v-if="t?.super_form_name">
              <p class="text-xs text-brand-dark-600 mb-1">Super Form</p>
              <p class="text-brand-dark-950">{{ t.super_form_name }}</p>
            </div>
          </div>

          <div v-if="t?.description">
            <p class="text-xs text-brand-dark-600 mb-1">Description</p>
            <p class="text-sm text-brand-dark-800 leading-relaxed">{{ t.description }}</p>
          </div>

          <div v-if="t?.notes">
            <p class="text-xs text-brand-dark-600 mb-1">Notes</p>
            <p class="text-sm text-brand-dark-800 leading-relaxed">{{ t.notes }}</p>
          </div>
        </div>

        <!-- Navigation footer -->
        <div class="border-t border-brand-cream-300 px-6 py-4 flex items-center justify-between shrink-0">
          <button
            :disabled="!hasPrev"
            class="flex items-center gap-1 px-3 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-30 disabled:cursor-not-allowed text-brand-dark-700 hover:bg-brand-cream-100 hover:text-brand-dark-950"
            @click="prev"
          >
            <ChevronLeft class="w-4 h-4" />
            Previous
          </button>
          <span class="text-xs text-brand-dark-600">
            {{ currentIndex + 1 }} / {{ traits.length }}
          </span>
          <button
            :disabled="!hasNext"
            class="flex items-center gap-1 px-3 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-30 disabled:cursor-not-allowed text-brand-dark-700 hover:bg-brand-cream-100 hover:text-brand-dark-950"
            @click="next"
          >
            Next
            <ChevronRight class="w-4 h-4" />
          </button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

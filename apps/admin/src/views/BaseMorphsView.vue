<script setup lang="ts">
import { ref, computed } from 'vue';
import { Edit2, Eye, AlertTriangle } from 'lucide-vue-next';
import { useTraits, useSpecies } from '@/composables/useGeckos';
import TraitEditSheet from '@/components/TraitEditSheet.vue';
import TraitDetailSheet from '@/components/TraitDetailSheet.vue';
import type { Trait, Species } from '@/types/gecko';
import type { InheritanceType } from '@/types/morph';

const { data: species } = useSpecies();
const selectedSpeciesId = ref<number | null>(null);

const { data: traits, isLoading } = useTraits(selectedSpeciesId);

const speciesById = computed<Record<number, Species>>(() =>
  Object.fromEntries((species.value ?? []).map((s) => [s.id, s])),
);

const sheetOpen = ref(false);
const editing = ref<Trait | null>(null);

const detailOpen = ref(false);
const viewing = ref<Trait | null>(null);

function openEdit(trait: Trait) {
  editing.value = trait;
  sheetOpen.value = true;
}

function openDetail(trait: Trait) {
  viewing.value = trait;
  detailOpen.value = true;
}

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
</script>

<template>
  <div>
    <!-- Species filter -->
    <div class="mb-4 flex items-center gap-3">
      <label class="text-sm font-medium text-brand-dark-700 shrink-0">Species</label>
      <select
        v-model="selectedSpeciesId"
        class="h-9 rounded-md border border-brand-cream-300 bg-white px-3 text-sm text-brand-dark-950 focus:outline-none focus:ring-2 focus:ring-brand-gold-500"
      >
        <option :value="null">All species</option>
        <option v-for="s in species" :key="s.id" :value="s.id">{{ s.common_name }}</option>
      </select>
    </div>

    <!-- Loading -->
    <div v-if="isLoading" class="text-brand-dark-600 text-sm py-8">Loading…</div>

    <!-- Table -->
    <div v-else class="rounded-xl border border-brand-cream-300 overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="bg-brand-cream-100 border-b border-brand-cream-300">
            <th class="text-left px-4 py-3 font-semibold text-brand-dark-700">Trait</th>
            <th class="text-left px-4 py-3 font-semibold text-brand-dark-700">Species</th>
            <th class="text-left px-4 py-3 font-semibold text-brand-dark-700">Code</th>
            <th class="text-left px-4 py-3 font-semibold text-brand-dark-700">Inheritance</th>
            <th class="text-left px-4 py-3 font-semibold text-brand-dark-700">Super Form</th>
            <th class="px-4 py-3" />
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="t in traits"
            :key="t.id"
            class="border-b border-brand-cream-200 last:border-0 hover:bg-brand-cream-50 transition-colors"
          >
            <td class="px-4 py-3 font-medium text-brand-dark-950">
              <div class="flex items-center gap-2">
                <img
                  v-if="t.example_photo_url"
                  :src="t.example_photo_url"
                  class="w-7 h-7 rounded object-cover border border-brand-cream-200 shrink-0"
                  alt=""
                />
                <span v-else class="w-7 h-7 shrink-0" />
                {{ t.trait_name }}
                <AlertTriangle
                  v-if="hasHealthWarning(t)"
                  class="w-3.5 h-3.5 text-amber-500 shrink-0"
                  :title="t.notes"
                />
              </div>
            </td>
            <td class="px-4 py-3 text-brand-dark-600 font-mono text-xs">
              {{ speciesById[t.species_id]?.code ?? '—' }}
            </td>
            <td class="px-4 py-3 text-brand-dark-600 font-mono text-xs">{{ t.trait_code || '—' }}</td>
            <td class="px-4 py-3">
              <span
                class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium"
                :class="INHERITANCE_BADGE[t.inheritance_type]?.class ?? ''"
              >
                {{ INHERITANCE_BADGE[t.inheritance_type]?.label ?? t.inheritance_type }}
              </span>
            </td>
            <td class="px-4 py-3 text-brand-dark-600 text-xs">{{ t.super_form_name || '—' }}</td>
            <td class="px-4 py-3">
              <div class="flex items-center gap-1">
                <button
                  :aria-label="`View ${t.trait_name}`"
                  class="p-1 text-brand-dark-500 hover:text-brand-dark-950 rounded"
                  @click="openDetail(t)"
                >
                  <Eye class="w-4 h-4" />
                </button>
                <button
                  :aria-label="`Edit ${t.trait_name}`"
                  class="p-1 text-brand-dark-500 hover:text-brand-dark-950 rounded"
                  @click="openEdit(t)"
                >
                  <Edit2 class="w-4 h-4" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <TraitEditSheet v-model:open="sheetOpen" :trait="editing" />
    <TraitDetailSheet
      v-model:open="detailOpen"
      v-model="viewing"
      :traits="traits ?? []"
      :species-by-id="speciesById"
    />
  </div>
</template>

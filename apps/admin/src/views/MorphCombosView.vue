<script setup lang="ts">
import { ref } from 'vue';
import { Plus, Edit2, Trash2, Dna } from 'lucide-vue-next';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useMorphCombos, useDeleteMorphCombo } from '@/composables/useMorphCombos';
import MorphComboFormSheet from '@/components/MorphComboFormSheet.vue';
import type { MorphCombo } from '@/types/morph';

const { data, isLoading } = useMorphCombos();
const { mutate: deleteCombo } = useDeleteMorphCombo();

const sheetOpen = ref(false);
const editing = ref<MorphCombo | null>(null);

function openCreate() {
  editing.value = null;
  sheetOpen.value = true;
}

function openEdit(combo: MorphCombo) {
  editing.value = combo;
  sheetOpen.value = true;
}

function confirmDelete(combo: MorphCombo) {
  if (confirm(`Delete "${combo.name}"?`)) {
    deleteCombo(combo.id);
  }
}
</script>

<template>
  <div class="px-4 sm:px-6 lg:px-8 py-8">
    <!-- Header -->
    <div class="flex items-center justify-between mb-8">
      <div>
        <h1 class="text-3xl font-display text-brand-dark-950">Morph Combos</h1>
        <p class="text-sm text-brand-dark-600 mt-1">
          Named combinations of base traits.
        </p>
      </div>
      <Button @click="openCreate">
        <Plus class="w-4 h-4 mr-2" />
        Add Combo
      </Button>
    </div>

    <!-- Loading -->
    <div v-if="isLoading" class="text-brand-dark-600 text-sm">Loading…</div>

    <!-- Empty -->
    <div
      v-else-if="!data?.combos?.length"
      class="text-center py-16 text-brand-dark-600"
    >
      <Dna class="w-10 h-10 mx-auto mb-3 text-brand-cream-400" />
      <p>No morph combos yet. Add one above.</p>
    </div>

    <!-- Grid -->
    <div
      v-else
      class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6"
    >
      <Card
        v-for="combo in data.combos"
        :key="combo.id"
        class="p-5 border-brand-cream-300 bg-brand-cream-50"
      >
        <div class="flex items-start justify-between mb-3">
          <div>
            <h3 class="font-semibold text-brand-dark-950">{{ combo.name }}</h3>
            <span
              v-if="combo.code"
              class="text-xs text-brand-dark-600 font-mono"
            >{{ combo.code }}</span>
          </div>
          <div class="flex gap-1 shrink-0 ml-2">
            <button
              class="p-1 text-brand-dark-600 hover:text-brand-dark-950"
              @click="openEdit(combo)"
            >
              <Edit2 class="w-4 h-4" />
            </button>
            <button
              class="p-1 text-brand-dark-600 hover:text-destructive"
              @click="confirmDelete(combo)"
            >
              <Trash2 class="w-4 h-4" />
            </button>
          </div>
        </div>

        <!-- Trait badges -->
        <div class="flex flex-wrap gap-1.5">
          <Badge
            v-for="req in combo.requirements"
            :key="req.trait_id"
            variant="outline"
            class="text-xs"
          >
            {{ req.trait_name }}
            <span class="ml-1 text-brand-dark-600">{{ req.required_zygosity }}</span>
          </Badge>
        </div>

        <p
          v-if="combo.description"
          class="mt-3 text-xs text-brand-dark-600 line-clamp-2"
        >
          {{ combo.description }}
        </p>
      </Card>
    </div>

    <MorphComboFormSheet v-model:open="sheetOpen" :combo="editing" />
  </div>
</template>

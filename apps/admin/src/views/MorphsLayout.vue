<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import { Dna } from 'lucide-vue-next';

const route = useRoute();
const tabs = [
  { name: 'morphs-base',   label: 'Traits' },
  { name: 'morphs-combos', label: 'Combos' },
];
const activeTab = computed(() => route.name);
</script>

<template>
  <div>
    <!-- Page header -->
    <div class="mb-6">
      <div class="flex items-center gap-2 mb-1">
        <Dna class="w-5 h-5 text-brand-gold-600" />
        <h1 class="text-3xl font-display text-brand-dark-950">Morphs</h1>
      </div>
      <!-- Tab nav -->
      <div class="flex gap-1 mt-4 border-b border-brand-cream-300">
        <RouterLink
          v-for="tab in tabs"
          :key="tab.name"
          :to="{ name: tab.name }"
          class="px-4 py-2 text-sm font-medium text-brand-dark-600 hover:text-brand-dark-950 border-b-2 border-transparent -mb-px transition-colors"
          active-class="!text-brand-gold-700 !border-brand-gold-600"
        >
          {{ tab.label }}
        </RouterLink>
      </div>
    </div>
    <!-- Sub-view -->
    <RouterView />
  </div>
</template>

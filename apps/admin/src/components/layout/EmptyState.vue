<script setup lang="ts">
import LowPolyGecko from '@/components/art/LowPolyGecko.vue';

defineProps<{ title: string; description?: string }>();
</script>

<template>
  <div
    class="flex flex-col items-center justify-center text-center gap-4 py-16 px-6 rounded-xl border border-dashed border-brand-cream-400 bg-brand-cream-50/60"
  >
    <LowPolyGecko class="opacity-90 animate-float" :size="180" />
    <div class="flex flex-col gap-1 max-w-md">
      <h3 class="font-serif text-2xl text-brand-dark-950">{{ title }}</h3>
      <p v-if="description" class="text-brand-dark-600 text-sm">{{ description }}</p>
    </div>
    <div class="mt-2">
      <slot name="actions" />
    </div>
  </div>
</template>

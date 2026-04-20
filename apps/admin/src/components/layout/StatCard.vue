<script setup lang="ts">
import type { Component } from 'vue';
import { Card } from '@/components/ui/card';
import LowPolyAccent from '@/components/art/LowPolyAccent.vue';

defineProps<{
  label: string;
  value: string | number;
  icon: Component;
  delta?: string;
  deltaTone?: 'up' | 'down' | 'neutral';
  hint?: string;
}>();
</script>

<template>
  <Card
    class="relative overflow-hidden border-brand-cream-300 bg-brand-cream-50 p-6 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg"
  >
    <div class="flex items-start justify-between gap-4">
      <div class="flex flex-col gap-1">
        <span class="text-sm font-medium text-brand-dark-600">{{ label }}</span>
        <span class="font-serif text-3xl text-brand-dark-950 leading-none mt-1">{{ value }}</span>
        <span v-if="hint" class="text-xs text-brand-dark-600 mt-1">{{ hint }}</span>
      </div>
      <div class="relative shrink-0">
        <div
          class="flex size-12 items-center justify-center rounded-xl bg-brand-gold-100 text-brand-gold-700 border border-brand-gold-200"
        >
          <component :is="icon" class="size-6" stroke-width="1.75" />
        </div>
        <LowPolyAccent class="absolute -top-1 -right-1" :size="18" />
      </div>
    </div>
    <div
      v-if="delta"
      class="mt-4 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium"
      :class="{
        'bg-brand-gold-100 text-brand-gold-800': deltaTone === 'up' || deltaTone === 'neutral' || !deltaTone,
        'bg-red-50 text-red-700': deltaTone === 'down',
      }"
    >
      {{ delta }}
    </div>
  </Card>
</template>

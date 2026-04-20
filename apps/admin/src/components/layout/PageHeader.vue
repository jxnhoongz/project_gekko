<script setup lang="ts">
defineProps<{ title: string; subtitle?: string; eyebrow?: string }>();
</script>

<template>
  <header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between mb-8">
    <div class="flex flex-col gap-1">
      <span
        v-if="eyebrow"
        class="text-xs uppercase tracking-[0.16em] text-brand-gold-700 font-semibold"
        >{{ eyebrow }}</span
      >
      <h1 class="font-serif text-3xl sm:text-4xl text-brand-dark-950 leading-tight">
        {{ title }}
      </h1>
      <p v-if="subtitle" class="text-brand-dark-600 text-sm sm:text-base max-w-2xl">
        {{ subtitle }}
      </p>
    </div>
    <div class="flex items-center gap-2">
      <slot name="actions" />
    </div>
  </header>
</template>

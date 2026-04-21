<script setup lang="ts">
withDefaults(
  defineProps<{
    size?: number;
    withText?: boolean;
    textClass?: string;
  }>(),
  { size: 32, withText: true, textClass: 'font-serif text-xl leading-none' },
);
</script>

<template>
  <div class="flex items-center gap-2">
    <img
      src="/logo/logo_low_poly.png"
      alt="Zenetic Gekkos"
      :width="size"
      :height="size"
      :style="{ width: `${size}px`, height: `${size}px` }"
      class="rounded-lg shrink-0 shadow-sm ring-1 ring-brand-cream-300 object-cover"
      loading="eager"
      decoding="async"
    />
    <span v-if="withText" :class="textClass">Zenetic</span>
  </div>
</template>

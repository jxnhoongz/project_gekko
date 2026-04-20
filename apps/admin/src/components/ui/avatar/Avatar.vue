<script setup lang="ts">
import { type HTMLAttributes, computed } from 'vue';
import { cn } from '@/lib/utils';

const props = defineProps<{ class?: HTMLAttributes['class'] }>();
const classes = computed(() =>
  cn(
    'relative flex size-9 shrink-0 overflow-hidden rounded-full bg-brand-gold-100 text-brand-gold-800 items-center justify-center font-semibold text-sm',
    props.class,
  ),
);
</script>

<template>
  <div :class="classes"><slot /></div>
</template>

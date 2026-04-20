<script setup lang="ts">
import { type HTMLAttributes, computed } from 'vue';
import { cn } from '@/lib/utils';
const props = defineProps<{ class?: HTMLAttributes['class'] }>();
const classes = computed(() =>
  cn('border-b border-brand-cream-200 transition-colors hover:bg-brand-cream-100/60', props.class),
);
</script>

<template>
  <tr :class="classes"><slot /></tr>
</template>

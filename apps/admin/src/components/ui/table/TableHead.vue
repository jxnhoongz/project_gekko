<script setup lang="ts">
import { type HTMLAttributes, computed } from 'vue';
import { cn } from '@/lib/utils';
const props = defineProps<{ class?: HTMLAttributes['class'] }>();
const classes = computed(() =>
  cn(
    'h-10 px-4 text-left align-middle font-medium text-brand-dark-600 text-xs uppercase tracking-wide',
    props.class,
  ),
);
</script>

<template>
  <th :class="classes"><slot /></th>
</template>

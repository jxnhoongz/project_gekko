<script setup lang="ts">
import { type HTMLAttributes, computed } from 'vue';
import { cn } from '@/lib/utils';

interface Props {
  orientation?: 'horizontal' | 'vertical';
  class?: HTMLAttributes['class'];
}
const props = withDefaults(defineProps<Props>(), { orientation: 'horizontal' });
const classes = computed(() =>
  cn(
    'shrink-0 bg-border',
    props.orientation === 'horizontal' ? 'h-px w-full' : 'w-px h-full',
    props.class,
  ),
);
</script>

<template>
  <div role="separator" :aria-orientation="orientation" :class="classes" />
</template>

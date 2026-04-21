<script setup lang="ts">
import { type HTMLAttributes, computed } from 'vue';
import { cn } from '@/lib/utils';
import { badgeVariants, type BadgeVariants } from '.';

interface Props {
  variant?: BadgeVariants['variant'];
  class?: HTMLAttributes['class'];
}
const props = defineProps<Props>();
const classes = computed(() => cn(badgeVariants({ variant: props.variant }), props.class));
</script>

<template>
  <span :class="classes"><slot /></span>
</template>

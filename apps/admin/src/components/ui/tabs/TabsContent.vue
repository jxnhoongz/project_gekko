<script setup lang="ts">
import { TabsContent, type TabsContentProps } from 'reka-ui';
import { type HTMLAttributes, computed } from 'vue';
import { cn } from '@/lib/utils';

const props = defineProps<TabsContentProps & { class?: HTMLAttributes['class'] }>();
const delegated = computed(() => {
  const { class: _c, ...rest } = props;
  return rest;
});
</script>

<template>
  <TabsContent v-bind="delegated" :class="cn('focus-visible:outline-none', props.class)">
    <slot />
  </TabsContent>
</template>

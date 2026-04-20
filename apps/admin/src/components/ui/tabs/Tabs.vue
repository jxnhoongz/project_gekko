<script setup lang="ts">
import { TabsRoot, type TabsRootProps, type TabsRootEmits, useForwardPropsEmits } from 'reka-ui';
import { type HTMLAttributes, computed } from 'vue';
import { cn } from '@/lib/utils';

const props = defineProps<TabsRootProps & { class?: HTMLAttributes['class'] }>();
const emits = defineEmits<TabsRootEmits>();
const delegated = computed(() => {
  const { class: _c, ...rest } = props;
  return rest;
});
const forwarded = useForwardPropsEmits(delegated, emits);
</script>

<template>
  <TabsRoot v-bind="forwarded" :class="cn('flex flex-col gap-4', props.class)">
    <slot />
  </TabsRoot>
</template>

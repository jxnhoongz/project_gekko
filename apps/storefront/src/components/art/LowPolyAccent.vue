<script setup lang="ts">
withDefaults(defineProps<{ class?: string; size?: number }>(), { size: 20 });
</script>

<template>
  <svg
    :class="$props.class"
    :width="size"
    :height="size"
    viewBox="0 0 20 20"
    fill="none"
    aria-hidden="true"
  >
    <polygon points="14,2 20,2 20,8" fill="#efc262" opacity="0.9" />
    <polygon points="20,8 20,2 14,2 14,8" fill="#e8a83a" opacity="0.0" />
    <polygon points="14,2 14,8 8,8"  fill="#d08b1f" opacity="0.8" />
    <polygon points="8,8 14,8 20,8"  fill="#b06c12" opacity="0.35" />
  </svg>
</template>

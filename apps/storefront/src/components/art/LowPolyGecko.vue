<script setup lang="ts">
withDefaults(defineProps<{ class?: string; size?: number }>(), { size: 220 });
</script>

<template>
  <svg
    :class="$props.class"
    :width="size"
    :height="size"
    viewBox="0 0 240 180"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden="true"
  >
    <g stroke="#d9cdac" stroke-width="0.6" stroke-linejoin="round">
      <!-- body facets (warm gold/cream, light from top-right) -->
      <polygon points="38,110 72,78 98,112"  fill="#f6dc9a" />
      <polygon points="72,78 108,74 98,112"  fill="#efc262" />
      <polygon points="108,74 138,82 98,112" fill="#e8a83a" />
      <polygon points="138,82 160,70 158,104" fill="#f6dc9a" />
      <polygon points="138,82 158,104 118,118" fill="#d08b1f" />
      <polygon points="98,112 118,118 84,140"  fill="#b06c12" />
      <polygon points="98,112 84,140 54,130"   fill="#8b5211" />
      <polygon points="98,112 54,130 38,110"   fill="#ad9a6f" />

      <!-- head -->
      <polygon points="158,104 186,90 200,106" fill="#efc262" />
      <polygon points="160,70 200,80 186,90"   fill="#fbefd0" />
      <polygon points="160,70 186,90 158,104"  fill="#e8a83a" />
      <polygon points="200,80 218,92 200,106"  fill="#f6dc9a" />
      <polygon points="200,80 200,106 186,90"  fill="#d08b1f" />

      <!-- eye (dark facet) -->
      <polygon points="196,88 204,86 200,94" fill="#110e0b" />

      <!-- tail curving down -->
      <polygon points="54,130 38,110 18,128" fill="#8f7e57" />
      <polygon points="18,128 38,110 30,150" fill="#726447" />
      <polygon points="30,150 54,130 46,162" fill="#564c37" />

      <!-- front leg -->
      <polygon points="118,118 130,138 102,144" fill="#8b5211" />
      <polygon points="102,144 130,138 120,158" fill="#6a3f10" />

      <!-- rear leg -->
      <polygon points="76,132 64,158 92,150" fill="#6a3f10" />
      <polygon points="64,158 92,150 80,168" fill="#4b2c0d" />
    </g>
  </svg>
</template>

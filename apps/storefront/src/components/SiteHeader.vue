<script setup lang="ts">
import { ref } from 'vue';
import { Button } from '@/components/ui/button';
import { Menu, X } from 'lucide-vue-next';
import BrandLogo from '@/components/BrandLogo.vue';

const mobileOpen = ref(false);
</script>

<template>
  <header class="sticky top-0 z-40 bg-brand-cream-50/95 backdrop-blur-sm border-b border-brand-cream-300">
    <div class="mx-auto max-w-6xl px-4 sm:px-6 h-16 flex items-center justify-between">
      <RouterLink :to="{ name: 'home' }" class="flex items-center">
        <BrandLogo :size="32" />
      </RouterLink>
      <!-- Desktop nav -->
      <nav class="hidden sm:flex items-center gap-6 text-sm">
        <RouterLink
          :to="{ name: 'geckos' }"
          class="text-brand-dark-700 hover:text-brand-dark-950"
          active-class="!text-brand-gold-700"
        >Geckos</RouterLink>
        <RouterLink
          :to="{ name: 'waitlist' }"
          active-class="!text-brand-gold-700"
        >
          <Button variant="default" size="sm">Join waitlist</Button>
        </RouterLink>
      </nav>
      <!-- Mobile toggle -->
      <Button
        class="sm:hidden"
        variant="ghost"
        size="icon"
        aria-label="Toggle menu"
        @click="mobileOpen = !mobileOpen"
      >
        <X v-if="mobileOpen" class="size-5" />
        <Menu v-else class="size-5" />
      </Button>
    </div>
    <!-- Mobile menu -->
    <div
      v-if="mobileOpen"
      class="sm:hidden border-t border-brand-cream-300 bg-brand-cream-50"
    >
      <div class="px-4 py-3 flex flex-col gap-2">
        <RouterLink
          :to="{ name: 'geckos' }"
          class="px-3 py-2 rounded-md hover:bg-brand-cream-100"
          @click="mobileOpen = false"
        >Geckos</RouterLink>
        <RouterLink
          :to="{ name: 'waitlist' }"
          class="px-3 py-2 rounded-md bg-brand-gold-600 text-white text-center hover:bg-brand-gold-700"
          @click="mobileOpen = false"
        >Join waitlist</RouterLink>
      </div>
    </div>
  </header>
</template>

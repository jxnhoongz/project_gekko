<script setup lang="ts">
import { useRouter } from 'vue-router';
import { Button } from '@/components/ui/button';
import { ArrowLeft } from 'lucide-vue-next';
import LowPolyGecko from '@/components/art/LowPolyGecko.vue';

const router = useRouter();
</script>

<template>
  <div
    class="min-h-screen flex flex-col items-center justify-center gap-6 bg-brand-cream-50 px-4 py-10 text-center"
  >
    <LowPolyGecko :size="200" class="animate-float" />
    <div class="flex flex-col gap-2 max-w-md">
      <span class="text-xs uppercase tracking-[0.16em] text-brand-gold-700 font-semibold"
        >Error 404</span
      >
      <h1 class="font-serif text-4xl">That page slipped away.</h1>
      <p class="text-brand-dark-600">
        This route isn't part of the admin — maybe it moved, maybe it's yet to be built.
      </p>
    </div>
    <Button variant="default" @click="router.push('/')">
      <ArrowLeft class="size-4" /> Back to dashboard
    </Button>
  </div>
</template>

<script setup lang="ts">
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { useRouter } from 'vue-router';
import SiteHeader from '@/components/SiteHeader.vue';
import SiteFooter from '@/components/SiteFooter.vue';
import PublicGeckoCard from '@/components/PublicGeckoCard.vue';
import { usePublicGeckos } from '@/composables/usePublicGeckos';

const router = useRouter();
const { data, isLoading, isError } = usePublicGeckos();
</script>

<template>
  <div class="min-h-screen flex flex-col bg-brand-cream-50 text-brand-dark-950">
    <SiteHeader />

    <main class="mx-auto max-w-6xl w-full px-4 sm:px-6 py-10 flex-1">
      <div class="flex flex-col gap-2 mb-8">
        <span class="text-xs uppercase tracking-wider text-brand-gold-700 font-semibold">Collection</span>
        <h1 class="font-serif text-4xl">Currently available</h1>
        <p class="text-brand-dark-700">Every gecko shown here is ready to ship (locally) or pickup in Phnom Penh.</p>
      </div>

      <div v-if="isLoading" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
        <Skeleton v-for="n in 6" :key="n" class="h-80 rounded-xl" />
      </div>

      <div
        v-else-if="isError"
        class="rounded-xl border border-red-200 bg-red-50 p-6 text-center text-red-900"
      >
        Couldn't load the collection. Please refresh.
      </div>

      <div
        v-else-if="data && data.geckos.length"
        class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5"
      >
        <PublicGeckoCard v-for="g in data.geckos" :key="g.code" :gecko="g" />
      </div>

      <div
        v-else
        class="rounded-xl border border-dashed border-brand-cream-400 bg-brand-cream-50 p-10 text-center flex flex-col items-center gap-4"
      >
        <p class="text-brand-dark-700">
          Nothing available right now — we'll announce new hatchlings to the waitlist first.
        </p>
        <Button @click="router.push({ name: 'waitlist' })">Join the waitlist</Button>
      </div>
    </main>

    <SiteFooter />
  </div>
</template>

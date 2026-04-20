<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { toast } from 'vue-sonner';
import { useAuthStore } from '@/stores/auth';
import AnimatedBackground from '@/components/AnimatedBackground.vue';
import LowPolyAccent from '@/components/art/LowPolyAccent.vue';

const email = ref('');
const password = ref('');
const auth = useAuthStore();
const router = useRouter();

async function onSubmit(e: Event) {
  e.preventDefault();
  const ok = await auth.login(email.value, password.value);
  if (ok) {
    toast.success('Welcome back.');
    router.push('/');
  } else {
    toast.error('Invalid email or password.');
  }
}
</script>

<template>
  <div class="relative min-h-screen flex items-center justify-center bg-brand-cream-50 px-4 py-10 overflow-hidden">
    <AnimatedBackground />

    <Card
      class="w-full max-w-md !p-0 !gap-0 border-brand-cream-300/70 bg-brand-cream-50/80 backdrop-blur-xl shadow-xl"
    >
      <div class="p-8 pb-6 flex flex-col gap-2">
        <div class="flex items-center gap-2">
          <LowPolyAccent :size="20" />
          <span class="text-xs font-semibold tracking-[0.16em] uppercase text-brand-gold-700">
            Zenetic Gekkos
          </span>
        </div>
        <h1 class="font-serif text-4xl text-brand-dark-950 leading-tight">Admin sign-in</h1>
        <p class="text-sm text-brand-dark-600">
          Sign in to manage the colony, waitlist and sales.
        </p>
      </div>

      <form class="px-8 pb-8 flex flex-col gap-5" @submit="onSubmit">
        <div class="flex flex-col gap-2">
          <Label for="email">Email</Label>
          <Input
            id="email"
            v-model="email"
            type="email"
            autocomplete="username"
            placeholder="you@zeneticgekkos.com"
            required
            class="bg-white"
          />
        </div>
        <div class="flex flex-col gap-2">
          <Label for="password">Password</Label>
          <Input
            id="password"
            v-model="password"
            type="password"
            autocomplete="current-password"
            required
            class="bg-white"
          />
        </div>
        <Button
          type="submit"
          size="lg"
          :disabled="auth.loading"
          class="mt-2"
        >
          {{ auth.loading ? 'Signing in…' : 'Sign in' }}
        </Button>
      </form>
    </Card>
  </div>
</template>

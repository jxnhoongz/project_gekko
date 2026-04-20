<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { toast } from 'vue-sonner';
import { useAuthStore } from '@/stores/auth';

const email = ref('');
const password = ref('');
const auth = useAuthStore();
const router = useRouter();

async function onSubmit(e: Event) {
  e.preventDefault();
  const ok = await auth.login(email.value, password.value);
  if (ok) {
    router.push('/');
  } else {
    toast.error('Invalid email or password.');
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-brand-cream-50 px-4">
    <Card class="w-full max-w-md border-brand-cream-300">
      <CardHeader>
        <CardTitle class="text-3xl">Zenetic Gekkos</CardTitle>
        <CardDescription class="text-brand-dark-600"> Admin sign-in </CardDescription>
      </CardHeader>
      <CardContent>
        <form class="flex flex-col gap-4" @submit="onSubmit">
          <div class="flex flex-col gap-2">
            <Label for="email">Email</Label>
            <Input id="email" v-model="email" type="email" autocomplete="username" required />
          </div>
          <div class="flex flex-col gap-2">
            <Label for="password">Password</Label>
            <Input
              id="password"
              v-model="password"
              type="password"
              autocomplete="current-password"
              required
            />
          </div>
          <Button
            type="submit"
            :disabled="auth.loading"
            class="bg-brand-gold-600 hover:bg-brand-gold-700 text-white"
          >
            {{ auth.loading ? 'Signing in…' : 'Sign in' }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>

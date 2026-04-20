<script setup lang="ts">
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { useAuthStore } from '@/stores/auth';

const auth = useAuthStore();

const stats = [
  { label: 'Total geckos', value: '—' },
  { label: 'Active pairings', value: '—' },
  { label: 'Eggs incubating', value: '—' },
  { label: 'Waitlist', value: '—' },
];
</script>

<template>
  <div class="flex flex-col gap-6">
    <div>
      <h1 class="text-4xl">Welcome back, {{ auth.admin?.name || auth.admin?.email }}.</h1>
      <p class="text-brand-dark-600 mt-2">
        Dashboard stats land here in phase 5. For now, this proves auth works.
      </p>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <Card v-for="stat in stats" :key="stat.label" class="border-brand-cream-300">
        <CardHeader>
          <CardTitle class="text-brand-dark-600 text-sm font-medium">
            {{ stat.label }}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div class="text-3xl font-serif">{{ stat.value }}</div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>

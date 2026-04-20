<script setup lang="ts">
import { useRouter } from 'vue-router';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { Avatar } from '@/components/ui/avatar';
import { LogOut } from 'lucide-vue-next';
import PageHeader from '@/components/layout/PageHeader.vue';
import { useAuthStore } from '@/stores/auth';
import { computed } from 'vue';

const auth = useAuthStore();
const router = useRouter();

const initials = computed(() => {
  const name = auth.admin?.name || auth.admin?.email || '';
  return name
    .split(/[\s@.]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0]?.toUpperCase())
    .join('') || 'ZG';
});

function onLogout() {
  auth.logout();
  router.push('/login');
}
</script>

<template>
  <div class="flex flex-col gap-8">
    <PageHeader
      eyebrow="Account"
      title="Settings"
      subtitle="Your admin profile and session controls."
    />

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <Card class="lg:col-span-2 border-brand-cream-300 bg-brand-cream-50 p-6 md:p-8 flex flex-col gap-6">
        <div class="flex items-center gap-4">
          <Avatar class="size-14 text-base">{{ initials }}</Avatar>
          <div class="flex flex-col">
            <div class="font-serif text-2xl">{{ auth.admin?.name || 'Admin' }}</div>
            <div class="text-sm text-brand-dark-600">{{ auth.admin?.email }}</div>
          </div>
        </div>
        <Separator />
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div class="flex flex-col gap-1.5">
            <Label>Display name</Label>
            <Input :model-value="auth.admin?.name || ''" disabled class="bg-white" />
          </div>
          <div class="flex flex-col gap-1.5">
            <Label>Email</Label>
            <Input :model-value="auth.admin?.email || ''" disabled class="bg-white" />
          </div>
        </div>
        <p class="text-xs text-brand-dark-600">
          Profile editing lands in a later phase. For now, these are read-only.
        </p>
      </Card>

      <Card class="border-brand-cream-300 bg-brand-cream-50 p-6 flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <h3 class="font-serif text-xl">Session</h3>
          <p class="text-sm text-brand-dark-600">
            Signing out clears your token on this device.
          </p>
        </div>
        <Button variant="secondary" class="!bg-brand-dark-950 !text-white hover:!bg-brand-dark-800" @click="onLogout">
          <LogOut class="size-4" />
          Log out
        </Button>
      </Card>
    </div>
  </div>
</template>

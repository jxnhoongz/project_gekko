<script setup lang="ts">
import { useRouter } from 'vue-router';
import { Button } from '@/components/ui/button';
import { LogOut, LayoutDashboard } from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const auth = useAuthStore();

function onLogout() {
  auth.logout();
  router.push('/login');
}
</script>

<template>
  <div class="min-h-screen flex bg-brand-cream-50 text-brand-dark-950">
    <aside class="w-56 border-r border-brand-cream-300 p-4 flex flex-col gap-2">
      <div class="font-serif text-xl mb-4">Zenetic</div>
      <RouterLink
        to="/"
        class="flex items-center gap-2 px-3 py-2 rounded-md hover:bg-brand-cream-100"
        active-class="bg-brand-gold-100 text-brand-gold-800"
      >
        <LayoutDashboard class="w-4 h-4" /> Dashboard
      </RouterLink>
    </aside>
    <div class="flex-1 flex flex-col">
      <header
        class="border-b border-brand-cream-300 px-6 py-3 flex items-center justify-between"
      >
        <div class="text-sm text-brand-dark-600">
          Signed in as {{ auth.admin?.email }}
        </div>
        <Button
          variant="ghost"
          size="sm"
          class="text-brand-dark-950 hover:bg-brand-cream-100"
          @click="onLogout"
        >
          <LogOut class="w-4 h-4 mr-2" /> Log out
        </Button>
      </header>
      <main class="p-6 flex-1 overflow-auto">
        <RouterView />
      </main>
    </div>
  </div>
</template>

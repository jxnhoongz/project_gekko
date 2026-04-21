<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { CheckCircle2, ArrowRight } from 'lucide-vue-next';
import SiteHeader from '@/components/SiteHeader.vue';
import SiteFooter from '@/components/SiteFooter.vue';
import LowPolyAccent from '@/components/art/LowPolyAccent.vue';
import { useWaitlistSignup } from '@/composables/useWaitlistSignup';
import { toast } from 'vue-sonner';

const route = useRoute();
const email = ref('');
const telegram = ref('');
const phone = ref('');
const interestedIn = ref('');
const notes = ref('');

const success = ref<'created' | 'deduplicated' | null>(null);
const mutation = useWaitlistSignup();

onMounted(() => {
  const q = route.query.interested_in;
  if (typeof q === 'string' && q) {
    interestedIn.value = q;
  }
});

async function onSubmit(e: Event) {
  e.preventDefault();
  if (!email.value.trim()) {
    toast.error('Email is required.');
    return;
  }
  try {
    const res = await mutation.mutateAsync({
      email: email.value.trim(),
      telegram: telegram.value.trim() || undefined,
      phone: phone.value.trim() || undefined,
      interested_in: interestedIn.value.trim() || undefined,
      notes: notes.value.trim() || undefined,
    });
    success.value = res.deduplicated ? 'deduplicated' : 'created';
  } catch (e: unknown) {
    const msg = (e as any)?.response?.data?.error ?? (e as Error).message ?? 'Something went wrong';
    toast.error(String(msg));
  }
}
</script>

<template>
  <div class="min-h-screen flex flex-col bg-brand-cream-50 text-brand-dark-950">
    <SiteHeader />

    <main class="mx-auto max-w-2xl w-full px-4 sm:px-6 py-12 flex-1">
      <!-- Success state -->
      <section
        v-if="success"
        class="rounded-xl border border-brand-cream-300 bg-brand-cream-50 p-10 text-center flex flex-col items-center gap-4"
      >
        <CheckCircle2 class="size-12 text-brand-gold-700" />
        <h1 class="font-serif text-3xl">
          {{ success === 'created' ? "You're on the list." : 'Already there.' }}
        </h1>
        <p class="text-brand-dark-700 max-w-md">
          {{
            success === 'created'
              ? "Thanks — we'll be in touch when new geckos are ready."
              : "It looks like you're already on the waitlist. We'll be in touch when new geckos are ready."
          }}
        </p>
        <Button variant="outline" @click="$router.push({ name: 'home' })">
          <ArrowRight class="size-4" /> Back to home
        </Button>
      </section>

      <!-- Form state -->
      <section v-else>
        <div class="mb-6">
          <span class="inline-flex items-center gap-2 text-xs font-semibold tracking-[0.16em] uppercase text-brand-gold-700">
            <LowPolyAccent :size="16" /> Waitlist
          </span>
          <h1 class="font-serif text-4xl mt-2">Tell us who to call.</h1>
          <p class="text-brand-dark-700 mt-2">
            We'll email you when geckos matching your interest become available. No spam — one note per drop.
          </p>
        </div>

        <form class="flex flex-col gap-5" @submit="onSubmit">
          <div class="flex flex-col gap-2">
            <Label for="wl-email">Email <span class="text-destructive">*</span></Label>
            <Input id="wl-email" v-model="email" type="email" required autocomplete="email" class="bg-white" />
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="flex flex-col gap-2">
              <Label for="wl-telegram">Telegram</Label>
              <Input id="wl-telegram" v-model="telegram" placeholder="@you" class="bg-white" />
            </div>
            <div class="flex flex-col gap-2">
              <Label for="wl-phone">Phone</Label>
              <Input id="wl-phone" v-model="phone" placeholder="+855…" class="bg-white" />
            </div>
          </div>
          <div class="flex flex-col gap-2">
            <Label for="wl-interest">Interested in (optional)</Label>
            <Input
              id="wl-interest"
              v-model="interestedIn"
              placeholder="Tangerine leopard, lilly white crested, etc."
              class="bg-white"
            />
          </div>
          <div class="flex flex-col gap-2">
            <Label for="wl-notes">Notes (optional)</Label>
            <textarea
              id="wl-notes"
              v-model="notes"
              rows="4"
              class="rounded-md border border-brand-cream-300 bg-white px-3 py-2 text-sm resize-y"
              placeholder="First-time keeper? Experienced? Preferred drop window?"
            />
          </div>
          <Button type="submit" size="lg" :disabled="mutation.isPending.value">
            {{ mutation.isPending.value ? 'Submitting…' : 'Join the waitlist' }}
          </Button>
        </form>
      </section>
    </main>

    <SiteFooter />
  </div>
</template>

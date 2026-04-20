<script setup lang="ts">
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { DollarSign } from 'lucide-vue-next';
import PageHeader from '@/components/layout/PageHeader.vue';
import EmptyState from '@/components/layout/EmptyState.vue';
</script>

<template>
  <div class="flex flex-col gap-8">
    <PageHeader
      eyebrow="Revenue"
      title="Sales"
      subtitle="Deposits, payouts, and handover status."
    >
      <template #actions>
        <Button size="sm" variant="default"><DollarSign class="size-4" /> Record sale</Button>
      </template>
    </PageHeader>

    <Card class="border-brand-cream-300 bg-brand-cream-50 p-6">
      <EmptyState
        title="No sales recorded yet."
        description="Pre-breeding phase — log your first sale once a gecko finds its home."
      />
    </Card>
  </div>
</template>

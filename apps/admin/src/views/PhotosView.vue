<script setup lang="ts">
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Upload } from 'lucide-vue-next';
import PageHeader from '@/components/layout/PageHeader.vue';
import EmptyState from '@/components/layout/EmptyState.vue';
</script>

<template>
  <div class="flex flex-col gap-8">
    <PageHeader
      eyebrow="Gallery"
      title="Photos"
      subtitle="One grid for everything — tag photos to geckos or clutches later."
    >
      <template #actions>
        <Button size="sm" variant="default"><Upload class="size-4" /> Upload</Button>
      </template>
    </PageHeader>

    <Card class="border-brand-cream-300 bg-brand-cream-50 p-6">
      <EmptyState
        title="No photos uploaded yet."
        description="Upload your first batch — growth shots, morphs, or marketing photos."
      >
        <template #actions>
          <Button variant="default" size="sm"><Upload class="size-4" /> Upload first photo</Button>
        </template>
      </EmptyState>
    </Card>
  </div>
</template>

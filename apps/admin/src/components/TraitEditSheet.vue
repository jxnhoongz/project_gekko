<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import {
  DialogRoot,
  DialogPortal,
  DialogOverlay,
  DialogContent,
} from 'reka-ui';
import { X, Upload, Clipboard, AlertTriangle } from 'lucide-vue-next';
import { toast } from 'vue-sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useUpdateTrait, useUploadTraitPhoto } from '@/composables/useGeckos';
import { INHERITANCE_TYPE_LABEL } from '@/types/morph';
import type { Trait } from '@/types/gecko';

const props = defineProps<{
  open: boolean;
  trait: Trait | null;
}>();
const emit = defineEmits<{ 'update:open': [value: boolean] }>();

const form = ref({
  trait_code: '',
  description: '',
  notes: '',
  inheritance_type: 'RECESSIVE' as string,
  super_form_name: '',
});

watch(
  () => [props.open, props.trait] as const,
  ([open, t]) => {
    if (open && t) {
      form.value = {
        trait_code: t.trait_code,
        description: t.description,
        notes: t.notes,
        inheritance_type: t.inheritance_type,
        super_form_name: t.super_form_name,
      };
    }
  },
  { immediate: true },
);

const { mutate: updateTrait, isPending: saving } = useUpdateTrait();
const { mutate: uploadPhoto, isPending: uploading } = useUploadTraitPhoto();

const photoPreview = computed(() => props.trait?.example_photo_url || null);
const fileInput = ref<HTMLInputElement | null>(null);

function close() {
  emit('update:open', false);
}

function submit() {
  if (!props.trait) return;
  updateTrait(
    { id: props.trait.id, payload: form.value },
    {
      onSuccess: close,
      onError: (e: any) =>
        toast.error(e?.response?.data?.error ?? 'Save failed'),
    },
  );
}


function doUpload(file: File) {
  if (!props.trait) return;
  uploadPhoto(
    { id: props.trait.id, file },
    { onError: (e: any) => toast.error(e?.response?.data?.error ?? 'Upload failed') },
  );
}

function onFileChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0];
  if (file) doUpload(file);
}

async function pasteFromClipboard() {
  try {
    const items = await navigator.clipboard.read();
    for (const item of items) {
      const imageType = item.types.find((t) => t.startsWith('image/'));
      if (imageType) {
        const blob = await item.getType(imageType);
        doUpload(new File([blob], 'pasted-image.png', { type: imageType }));
        return;
      }
    }
    toast.error('No image found in clipboard');
  } catch {
    toast.error('Could not read clipboard');
  }
}

function onDocumentPaste(e: ClipboardEvent) {
  if (!props.open || !props.trait) return;
  const file = Array.from(e.clipboardData?.files ?? []).find((f) =>
    f.type.startsWith('image/'),
  );
  if (!file) return;
  e.preventDefault();
  doUpload(file);
}

onMounted(() => document.addEventListener('paste', onDocumentPaste));
onUnmounted(() => document.removeEventListener('paste', onDocumentPaste));

const isHealthWarning = computed(() =>
  props.trait
    ? ['Enigma', 'Lemon Frost'].some((n) => props.trait!.trait_name.includes(n))
    : false,
);

const inheritanceTypes = ['RECESSIVE', 'CO_DOMINANT', 'DOMINANT', 'POLYGENIC'];
</script>

<template>
  <DialogRoot :open="open" @update:open="emit('update:open', $event)">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 bg-brand-dark-950/40 z-40" />
      <DialogContent
        class="fixed right-0 top-0 h-full w-full max-w-lg bg-brand-cream-50 border-l border-brand-cream-300 shadow-xl z-50 flex flex-col overflow-y-auto focus:outline-none"
      >
        <!-- Header -->
        <div class="flex items-center justify-between px-6 py-4 border-b border-brand-cream-300">
          <div>
            <h2 class="text-xl font-semibold text-brand-dark-950">
              {{ trait?.trait_name ?? '—' }}
            </h2>
            <p v-if="trait?.species_id" class="text-xs text-brand-dark-600 mt-0.5">
              Base Morph
            </p>
          </div>
          <button aria-label="Close" class="text-brand-dark-600 hover:text-brand-dark-950" @click="close">
            <X class="w-5 h-5" />
          </button>
        </div>

        <!-- Health warning -->
        <div
          v-if="isHealthWarning"
          class="mx-6 mt-4 flex gap-2 rounded-lg bg-amber-50 border border-amber-200 px-4 py-3 text-sm text-amber-800"
        >
          <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5" />
          <p>{{ trait?.notes }}</p>
        </div>

        <!-- Body -->
        <div class="flex-1 px-6 py-6 space-y-5">

          <!-- Photo -->
          <div class="space-y-2">
            <Label>Example Photo</Label>
            <div class="flex items-start gap-4">
              <div
                class="w-24 h-24 rounded-lg border border-brand-cream-300 bg-brand-cream-100 flex items-center justify-center overflow-hidden shrink-0"
              >
                <img
                  v-if="photoPreview"
                  :src="photoPreview"
                  class="w-full h-full object-cover"
                  alt="Trait photo"
                />
                <span v-else class="text-xs text-brand-dark-500">No photo</span>
              </div>
              <div class="flex flex-col gap-2 pt-1">
                <Button variant="outline" size="sm" :disabled="uploading" @click="fileInput?.click()">
                  <Upload class="w-4 h-4 mr-1.5" />
                  {{ uploading ? 'Uploading…' : 'Upload photo' }}
                </Button>
                <Button variant="outline" size="sm" :disabled="uploading" @click="pasteFromClipboard">
                  <Clipboard class="w-4 h-4 mr-1.5" />
                  Paste from clipboard
                </Button>
                <p class="text-xs text-brand-dark-600">JPG, PNG, WebP or GIF. Max 10 MB.</p>
              </div>
            </div>
            <input
              ref="fileInput"
              type="file"
              accept="image/jpeg,image/png,image/webp,image/gif"
              class="hidden"
              @change="onFileChange"
            />
          </div>

          <!-- Inheritance type -->
          <div class="space-y-1.5">
            <Label>Inheritance Type</Label>
            <select
              v-model="form.inheritance_type"
              class="w-full rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500"
            >
              <option v-for="t in inheritanceTypes" :key="t" :value="t">
                {{ INHERITANCE_TYPE_LABEL[t as keyof typeof INHERITANCE_TYPE_LABEL] }}
              </option>
            </select>
          </div>

          <!-- Super form name (CO_DOMINANT only) -->
          <div v-if="form.inheritance_type === 'CO_DOMINANT'" class="space-y-1.5">
            <Label>Super Form Name</Label>
            <Input v-model="form.super_form_name" placeholder="e.g. Super Snow" />
          </div>

          <!-- Trait code -->
          <div class="space-y-1.5">
            <Label>Code</Label>
            <Input v-model="form.trait_code" placeholder="e.g. TREM" />
          </div>

          <!-- Description -->
          <div class="space-y-1.5">
            <Label>Description</Label>
            <textarea
              v-model="form.description"
              rows="3"
              class="w-full rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500 resize-none"
            />
          </div>

          <!-- Notes (internal) -->
          <div class="space-y-1.5">
            <Label>Notes <span class="text-xs text-brand-dark-600">(internal)</span></Label>
            <textarea
              v-model="form.notes"
              rows="3"
              class="w-full rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500 resize-none"
            />
          </div>
        </div>

        <!-- Footer -->
        <div class="px-6 py-4 border-t border-brand-cream-300 flex justify-end gap-3">
          <Button variant="ghost" @click="close">Cancel</Button>
          <Button :disabled="saving" @click="submit">
            {{ saving ? 'Saving…' : 'Save Changes' }}
          </Button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import {
  DialogRoot,
  DialogPortal,
  DialogOverlay,
  DialogContent,
} from 'reka-ui';
import { X, Plus, Trash2 } from 'lucide-vue-next';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { useSpecies, useTraits } from '@/composables/useGeckos';
import {
  useCreateMorphCombo,
  useUpdateMorphCombo,
} from '@/composables/useMorphCombos';
import type { MorphCombo, MorphComboTraitInput } from '@/types/morph';

const props = defineProps<{
  open: boolean;
  combo: MorphCombo | null;
}>();
const emit = defineEmits<{
  'update:open': [value: boolean];
}>();

// useSpecies returns Species[] directly (data.value is Species[])
const { data: speciesData } = useSpecies();
// useTraits returns Trait[] directly (data.value is Trait[])
const { data: traitsData } = useTraits();

const allTraits = computed(() => traitsData.value ?? []);

const form = ref({
  species_id: 0,
  name: '',
  code: '',
  description: '',
  notes: '',
  example_photo_url: '',
  requirements: [] as MorphComboTraitInput[],
});

const addTraitID = ref<number | null>(null);
const addZygosity = ref<'HOM' | 'HET' | 'POSS_HET'>('HOM');

watch(
  () => props.combo,
  (c) => {
    if (c) {
      form.value = {
        species_id: c.species_id,
        name: c.name,
        code: c.code,
        description: c.description,
        notes: c.notes,
        example_photo_url: c.example_photo_url,
        requirements: c.requirements.map((r) => ({
          trait_id: r.trait_id,
          required_zygosity: r.required_zygosity,
        })),
      };
    } else {
      form.value = {
        species_id:
          speciesData.value?.find((s) => s.code === 'LP')?.id ?? 0,
        name: '',
        code: '',
        description: '',
        notes: '',
        example_photo_url: '',
        requirements: [],
      };
    }
  },
  { immediate: true },
);

const { mutate: createCombo, isPending: creating } = useCreateMorphCombo();
const { mutate: updateCombo, isPending: updating } = useUpdateMorphCombo();
const saving = computed(() => creating.value || updating.value);

function addRequirement() {
  if (!addTraitID.value) return;
  if (form.value.requirements.some((r) => r.trait_id === addTraitID.value))
    return;
  form.value.requirements.push({
    trait_id: addTraitID.value,
    required_zygosity: addZygosity.value,
  });
  addTraitID.value = null;
}

function removeRequirement(index: number) {
  form.value.requirements.splice(index, 1);
}

function traitNameFor(id: number) {
  return allTraits.value.find((t) => t.id === id)?.trait_name ?? `#${id}`;
}

function close() {
  emit('update:open', false);
}

function submit() {
  const payload = { ...form.value };
  if (props.combo) {
    updateCombo({ id: props.combo.id, payload }, { onSuccess: close });
  } else {
    createCombo(payload, { onSuccess: close });
  }
}
</script>

<template>
  <DialogRoot :open="open" @update:open="emit('update:open', $event)">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 bg-brand-dark-950/40 z-40" />
      <DialogContent
        class="fixed right-0 top-0 h-full w-full max-w-lg bg-brand-cream-50 border-l border-brand-cream-300 shadow-xl z-50 flex flex-col overflow-y-auto focus:outline-none"
      >
        <!-- Header -->
        <div
          class="flex items-center justify-between px-6 py-4 border-b border-brand-cream-300"
        >
          <h2 class="text-xl font-semibold text-brand-dark-950">
            {{ combo ? 'Edit Combo' : 'New Morph Combo' }}
          </h2>
          <button
            class="text-brand-dark-600 hover:text-brand-dark-950"
            @click="close"
          >
            <X class="w-5 h-5" />
          </button>
        </div>

        <!-- Body -->
        <div class="flex-1 px-6 py-6 space-y-5">
          <!-- Species -->
          <div class="space-y-1.5">
            <Label>Species</Label>
            <select
              v-model="form.species_id"
              :disabled="!!combo"
              class="w-full rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500"
            >
              <option
                v-for="s in speciesData"
                :key="s.id"
                :value="s.id"
              >
                {{ s.common_name }}
              </option>
            </select>
          </div>

          <!-- Name + Code -->
          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <Label>Name <span class="text-destructive ml-0.5">*</span></Label>
              <Input v-model="form.name" placeholder="Raptor" />
            </div>
            <div class="space-y-1.5">
              <Label>Code</Label>
              <Input v-model="form.code" placeholder="RAPT" />
            </div>
          </div>

          <!-- Description -->
          <div class="space-y-1.5">
            <Label>Description</Label>
            <textarea
              v-model="form.description"
              rows="2"
              class="w-full rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500 resize-none"
            />
          </div>

          <!-- Notes (internal) -->
          <div class="space-y-1.5">
            <Label>Notes <span class="text-xs text-brand-dark-600">(internal)</span></Label>
            <textarea
              v-model="form.notes"
              rows="2"
              class="w-full rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500 resize-none"
            />
          </div>

          <!-- Requirements -->
          <div class="space-y-2">
            <Label>Required Traits</Label>
            <div
              v-if="form.requirements.length"
              class="space-y-1.5"
            >
              <div
                v-for="(req, i) in form.requirements"
                :key="req.trait_id"
                class="flex items-center justify-between bg-brand-cream-100 rounded-lg px-3 py-2"
              >
                <span class="text-sm font-medium text-brand-dark-950">
                  {{ traitNameFor(req.trait_id) }}
                </span>
                <div class="flex items-center gap-2">
                  <Badge variant="outline" class="text-xs">
                    {{ req.required_zygosity }}
                  </Badge>
                  <button
                    class="text-brand-dark-600 hover:text-destructive"
                    @click="removeRequirement(i)"
                  >
                    <Trash2 class="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
            <p v-else class="text-sm text-brand-dark-600">No traits added yet.</p>

            <!-- Add trait row -->
            <div class="flex gap-2 mt-2">
              <select
                v-model="addTraitID"
                class="flex-1 rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500"
              >
                <option :value="null">Select trait…</option>
                <option
                  v-for="t in allTraits.filter(
                    (t) =>
                      t.species_id === form.species_id &&
                      !form.requirements.some((r) => r.trait_id === t.id),
                  )"
                  :key="t.id"
                  :value="t.id"
                >
                  {{ t.trait_name }}
                </option>
              </select>
              <select
                v-model="addZygosity"
                class="rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500"
              >
                <option value="HOM">HOM</option>
                <option value="HET">HET</option>
                <option value="POSS_HET">POSS HET</option>
              </select>
              <Button variant="outline" size="sm" @click="addRequirement">
                <Plus class="w-4 h-4" />
              </Button>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div
          class="px-6 py-4 border-t border-brand-cream-300 flex justify-end gap-3"
        >
          <Button variant="ghost" @click="close">Cancel</Button>
          <Button :disabled="saving || !form.name" @click="submit">
            {{ saving ? 'Saving…' : combo ? 'Save Changes' : 'Create Combo' }}
          </Button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<!-- [cn-fork] Admin Feature Modules Management View -->
<!-- eslint-disable vue/multi-word-component-names -->
<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading">
        <div class="space-y-6 w-full">
          <div
            v-for="item in featureList"
            :key="item.key"
            class="p-4 rounded-lg border bg-card text-card-foreground shadow-sm flex items-start justify-between gap-4"
          >
            <div class="space-y-1">
              <h3 class="text-base font-semibold leading-none tracking-tight">
                {{ $t(`admin.features.${item.key}.title`) }}
              </h3>
              <p class="text-sm text-muted-foreground">
                {{ $t(`admin.features.${item.key}.description`) }}
              </p>
            </div>
            <div class="flex items-center shrink-0 pt-1">
              <Switch
                :checked="Boolean(featureStore.features[item.key])"
                :disabled="updatingKey === item.key"
                @update:checked="(val) => onToggleFeature(item.key, val)"
              />
            </div>
          </div>
        </div>
      </LoadingOverlay>
    </template>
    <template #help>
      <p>{{ $t('admin.features.help') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
defineOptions({
  name: 'AdminFeatures'
})
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingOverlay from '@/components/layout/LoadingOverlay.vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import { Switch } from '@shared-ui/components/ui/switch'
import { useFeatureStore } from '@/stores/feature'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http'

const { t } = useI18n()
const emitter = useEmitter()
const featureStore = useFeatureStore()
const isLoading = ref(false)
const updatingKey = ref(null)

const featureList = [
  { key: 'livechat' },
  { key: 'helpcenter' },
  { key: 'ai' },
  { key: 'csat' },
  { key: 'email_channel' },
  { key: 'user_chat' }
]

onMounted(async () => {
  isLoading.value = true
  try {
    await featureStore.fetchFeatures()
  } finally {
    isLoading.value = false
  }
})

const onToggleFeature = async (key, checked) => {
  updatingKey.value = key
  try {
    await featureStore.updateFeature(key, checked)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    updatingKey.value = null
  }
}
</script>

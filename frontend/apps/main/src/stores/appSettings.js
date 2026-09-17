import { defineStore } from 'pinia'
import api from '@/api'

export const useAppSettingsStore = defineStore('settings', {
    state: () => ({
        settings: {},
        public_config: {}
    }),
    actions: {
        async fetchSettings (key = 'general') {
            try {
                const response = await api.getSettings(key)
                this.settings = response?.data?.data || {}
                // [cn-fork] sync features to useFeatureStore if present
                if (this.settings.features) {
                    const { useFeatureStore } = await import('@/stores/feature')
                    useFeatureStore().setFeatures(this.settings.features)
                }
                return this.settings
            } catch (error) {
                // Pass
            }
        },
        async fetchPublicConfig () {
            try {
                const response = await api.getConfig()
                this.public_config = response?.data?.data || {}
                return this.public_config
            } catch (error) {
                // Pass
            }
        },
        setSettings (newSettings) {
            this.settings = newSettings
        },
        setPublicConfig (newPublicConfig) {
            this.public_config = newPublicConfig
        }
    }
})

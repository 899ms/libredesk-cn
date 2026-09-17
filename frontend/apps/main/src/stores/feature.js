// [cn-fork] Pinia store for runtime feature toggles
import { defineStore } from 'pinia'
import api from '@/api'

export const useFeatureStore = defineStore('features', {
  state: () => ({
    features: {
      livechat: true,
      helpcenter: false,
      ai: false,
      csat: true,
      email_channel: false,
      user_chat: false
    },
    loaded: false
  }),

  getters: {
    isEnabled: (state) => (featureName) => {
      return Boolean(state.features[featureName])
    }
  },

  actions: {
    setFeatures (features) {
      if (features && typeof features === 'object') {
        this.features = { ...this.features, ...features }
        this.loaded = true
      }
    },

    async fetchFeatures () {
      try {
        const response = await api.getSettings('general')
        const data = response?.data?.data || {}
        if (data.features) {
          this.setFeatures(data.features)
        }
        return this.features
      } catch (error) {
        // Fall back to current state
        return this.features
      }
    },

    async updateFeature (name, enabled) {
      const updated = {
        ...this.features,
        [name]: enabled
      }
      // Call PUT /api/v1/settings/general with { features: { [name]: enabled } }
      await api.updateSettings('general', {
        features: {
          [name]: enabled
        }
      })
      this.features = updated
      return this.features
    }
  }
})

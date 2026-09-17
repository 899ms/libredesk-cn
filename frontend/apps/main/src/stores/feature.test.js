import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useFeatureStore } from './feature'
import { filterNavItems } from '../utils/nav-permissions'

vi.mock('@/api', () => ({
  default: {
    getSettings: vi.fn(),
    updateSettings: vi.fn()
  }
}))

describe('useFeatureStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('initializes with default features', () => {
    const store = useFeatureStore()
    expect(store.features.livechat).toBe(true)
    expect(store.features.helpcenter).toBe(false)
    expect(store.features.ai).toBe(false)
    expect(store.features.csat).toBe(true)
    expect(store.features.email_channel).toBe(false)
    expect(store.features.user_chat).toBe(false)
  })

  it('isEnabled returns correct boolean value', () => {
    const store = useFeatureStore()
    expect(store.isEnabled('livechat')).toBe(true)
    expect(store.isEnabled('ai')).toBe(false)
    expect(store.isEnabled('non_existent')).toBe(false)
  })

  it('setFeatures merges features into state', () => {
    const store = useFeatureStore()
    store.setFeatures({ ai: true, livechat: false })
    expect(store.features.ai).toBe(true)
    expect(store.features.livechat).toBe(false)
    expect(store.features.csat).toBe(true)
    expect(store.loaded).toBe(true)
  })

  it('filterNavItems filters out items and groups when feature is disabled', () => {
    const store = useFeatureStore()
    const navItems = [
      {
        titleKey: 'general',
        href: '/admin/general'
      },
      {
        titleKey: 'helpCenter',
        feature: 'helpcenter',
        children: [
          {
            titleKey: 'helpCenterList',
            href: '/admin/help-center',
            feature: 'helpcenter'
          }
        ]
      },
      {
        titleKey: 'ai',
        feature: 'ai',
        children: [
          {
            titleKey: 'aiProviders',
            href: '/admin/ai/providers'
          }
        ]
      }
    ]

    const filtered = filterNavItems(navItems, () => true, store.isEnabled)
    expect(filtered).toHaveLength(1)
    expect(filtered[0].titleKey).toBe('general')

    // Now enable ai
    store.setFeatures({ ai: true })
    const filteredWithAi = filterNavItems(navItems, () => true, store.isEnabled)
    expect(filteredWithAi).toHaveLength(2)
    expect(filteredWithAi.map(i => i.titleKey)).toEqual(['general', 'ai'])
  })
})

// [cn-fork] filterNavItems supports optional isFeatureEnabled check
export const filterNavItems = (navItems, can, isFeatureEnabled) => {
    return navItems
        .map(item => {
            // Check feature toggle first if provided
            if (item.feature && isFeatureEnabled && !isFeatureEnabled(item.feature)) {
                return null
            }
            // Process children first
            const filteredChildren = item.children
                ? filterNavItems(item.children, can, isFeatureEnabled)
                : undefined
            // Check item's permission
            const hasAccess = item.permission ? can(item.permission) : true
            // Only keep the item if:
            // 1. Has required permission (or none required)
            // 2. Has valid children (if parent item)
            const keep = hasAccess && (!item.children || filteredChildren.length > 0)
            return keep ? { ...item, children: filteredChildren } : null
        })
        .filter(Boolean) // Remove null entries
}
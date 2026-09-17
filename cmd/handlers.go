package main

import (
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/ws"
	// [cn-fork]
	"github.com/abhinavxd/libredesk/internal/feature"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	maxPageSize = 500
	maxIDsParam = 200
)

// initHandlers initializes the HTTP routes and handlers for the application.
func initHandlers(g *fastglue.Fastglue, hub *ws.Hub) {
	// getAndHead registers both methods: uptime checkers and link validators probe with HEAD.
	getAndHead := func(path string, h fastglue.FastRequestHandler) {
		g.GET(path, h)
		g.HEAD(path, h)
	}

	// Authentication.
	g.POST("/api/v1/auth/login", rateLimit(handleLogin, "auth"))
	g.GET("/logout", auth(handleLogout))
	g.GET("/api/v1/oidc/{id}/login", rateLimit(handleOIDCLogin, "auth"))
	g.GET("/api/v1/oidc/{id}/finish", rateLimit(handleOIDCCallback, "auth"))

	// i18n.
	g.GET("/api/v1/lang", handleGetAvailableLanguages)
	g.GET("/api/v1/lang/{lang}", handleGetI18nLang)

	// Public config for app initialization.
	g.GET("/api/v1/config", handleGetConfig)

	// Media - supports both authenticated access and signed URLs.
	g.GET("/uploads/{uuid}", authOrSignedURL(handleServeMedia))
	g.POST("/api/v1/media", auth(handleMediaUpload))

	// Settings.
	g.GET("/api/v1/settings/general", auth(handleGetGeneralSettings))
	g.PUT("/api/v1/settings/general", perm(clearsHCCache(handleUpdateGeneralSettings), "general_settings:manage"))
	g.GET("/api/v1/settings/notifications/email", perm(handleGetEmailNotificationSettings, "notification_settings:manage"))
	g.PUT("/api/v1/settings/notifications/email", perm(handleUpdateEmailNotificationSettings, "notification_settings:manage"))

	// OpenID connect single sign-on.
	g.GET("/api/v1/oidc", perm(handleGetAllOIDC, "oidc:manage"))
	g.POST("/api/v1/oidc", perm(handleCreateOIDC, "oidc:manage"))
	g.GET("/api/v1/oidc/{id}", perm(handleGetOIDC, "oidc:manage"))
	g.PUT("/api/v1/oidc/{id}", perm(handleUpdateOIDC, "oidc:manage"))
	g.DELETE("/api/v1/oidc/{id}", perm(handleDeleteOIDC, "oidc:manage"))

	// Conversations.
	g.GET("/api/v1/conversations/all", perm(handleGetAllConversations, "conversations:read_all"))
	g.GET("/api/v1/conversations/unassigned", perm(handleGetUnassignedConversations, "conversations:read_unassigned"))
	g.GET("/api/v1/conversations/assigned", perm(handleGetAssignedConversations, "conversations:read_assigned"))
	g.GET("/api/v1/conversations/mentioned", perm(handleGetMentionedConversations, "conversations:read"))
	g.GET("/api/v1/conversations/sidebar-counts", perm(handleGetSidebarCounts, "conversations:read"))
	g.GET("/api/v1/teams/{id}/conversations/unassigned", perm(handleGetTeamUnassignedConversations, "conversations:read_team_inbox"))
	g.GET("/api/v1/views/{id}/conversations", perm(handleGetViewConversations, "conversations:read"))
	g.GET("/api/v1/views/{id}/count", perm(handleGetViewCount, "conversations:read"))
	g.GET("/api/v1/conversations/{uuid}", perm(handleGetConversation, "conversations:read"))
	g.GET("/api/v1/conversations/{uuid}/participants", perm(handleGetConversationParticipants, "conversations:read"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/user", perm(handleUpdateUserAssignee, "conversations:update_user_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/team", perm(handleUpdateTeamAssignee, "conversations:update_team_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/user/remove", perm(handleRemoveUserAssignee, "conversations:update_user_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/team/remove", perm(handleRemoveTeamAssignee, "conversations:update_team_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/priority", perm(handleUpdateConversationPriority, "conversations:update_priority"))
	g.PUT("/api/v1/conversations/{uuid}/status", perm(handleUpdateConversationStatus, "conversations:update_status"))
	g.PUT("/api/v1/conversations/{uuid}/last-seen", perm(handleUpdateConversationAssigneeLastSeen, "conversations:read"))
	g.PUT("/api/v1/conversations/{uuid}/mark-unread", perm(handleMarkConversationAsUnread, "conversations:read"))
	g.POST("/api/v1/conversations/{uuid}/tags", perm(handleUpdateConversationtags, "conversations:update_tags"))
	g.GET("/api/v1/conversations/{uuid}/page-visits", perm(handleGetContactPageVisits, "conversations:read"))
	g.GET("/api/v1/conversations/{cuuid}/messages/{uuid}", perm(handleGetMessage, "messages:read"))
	g.GET("/api/v1/conversations/{uuid}/messages", perm(handleGetMessages, "messages:read"))
	g.GET("/api/v1/conversations/{uuid}/transcript", perm(handleDownloadConversationTranscript, "messages:read"))
	g.POST("/api/v1/conversations/{cuuid}/messages", auth(handleSendMessage))
	g.PUT("/api/v1/conversations/{cuuid}/messages/{uuid}/retry", perm(handleRetryMessage, "messages:write"))
	g.DELETE("/api/v1/conversations/{cuuid}/messages/{uuid}", perm(handleDeleteMessage, "messages:write_private"))
	g.POST("/api/v1/conversations", perm(handleCreateConversation, "conversations:write"))
	g.PUT("/api/v1/conversations/{uuid}/custom-attributes", auth(handleUpdateConversationCustomAttributes))
	g.PUT("/api/v1/conversations/{uuid}/contacts/custom-attributes", auth(handleUpdateContactCustomAttributes))
	// Draft endpoints
	g.GET("/api/v1/drafts", auth(handleGetAllDrafts))
	g.POST("/api/v1/conversations/{uuid}/draft", auth(handleUpsertConversationDraft))
	g.DELETE("/api/v1/conversations/{uuid}/draft", auth(handleDeleteConversationDraft))

	// Search.
	g.GET("/api/v1/conversations/search", perm(handleSearchConversations, "conversations:read"))
	g.GET("/api/v1/messages/search", perm(handleSearchMessages, "messages:read"))
	g.GET("/api/v1/contacts/search", perm(handleSearchContacts, "contacts:read"))

	// Views.
	g.GET("/api/v1/views/me", perm(handleGetUserViews, "view:manage"))
	g.POST("/api/v1/views/me", perm(handleCreateUserView, "view:manage"))
	g.PUT("/api/v1/views/me/{id}", perm(handleUpdateUserView, "view:manage"))
	g.DELETE("/api/v1/views/me/{id}", perm(handleDeleteUserView, "view:manage"))

	g.GET("/api/v1/views/shared", auth(handleGetSharedViews))

	g.GET("/api/v1/shared-views", perm(handleGetAllSharedViews, "shared_views:manage"))
	g.GET("/api/v1/shared-views/{id}", perm(handleGetSharedView, "shared_views:manage"))
	g.POST("/api/v1/shared-views", perm(handleCreateSharedView, "shared_views:manage"))
	g.PUT("/api/v1/shared-views/{id}", perm(handleUpdateSharedView, "shared_views:manage"))
	g.DELETE("/api/v1/shared-views/{id}", perm(handleDeleteSharedView, "shared_views:manage"))

	// Status and priority.
	g.GET("/api/v1/statuses", auth(handleGetStatuses))
	g.POST("/api/v1/statuses", perm(handleCreateStatus, "status:manage"))
	g.PUT("/api/v1/statuses/{id}", perm(handleUpdateStatus, "status:manage"))
	g.DELETE("/api/v1/statuses/{id}", perm(handleDeleteStatus, "status:manage"))
	g.GET("/api/v1/priorities", auth(handleGetPriorities))

	// Tags.
	g.GET("/api/v1/tags", auth(handleGetTags))
	g.POST("/api/v1/tags", perm(handleCreateTag, "tags:manage"))
	g.PUT("/api/v1/tags/{id}", perm(handleUpdateTag, "tags:manage"))
	g.DELETE("/api/v1/tags/{id}", perm(handleDeleteTag, "tags:manage"))
	g.POST("/api/v1/tags/import", perm(handleImportTags, "tags:manage"))
	g.GET("/api/v1/tags/import/status", perm(handleGetTagImportStatus, "tags:manage"))

	// Macros.
	g.GET("/api/v1/macros", auth(handleGetMacros))
	g.GET("/api/v1/macros/compact", perm(handleGetMacrosCompact, "macros:manage"))
	g.GET("/api/v1/macros/search", auth(handleSearchMacros))
	g.GET("/api/v1/macros/{id}", auth(handleGetMacro))
	g.POST("/api/v1/macros", perm(handleCreateMacro, "macros:manage"))
	g.PUT("/api/v1/macros/{id}", perm(handleUpdateMacro, "macros:manage"))
	g.DELETE("/api/v1/macros/{id}", perm(handleDeleteMacro, "macros:manage"))
	g.POST("/api/v1/conversations/{uuid}/macros/{id}/apply", auth(handleApplyMacro))

	// Agents.
	g.GET("/api/v1/agents/me", auth(handleGetCurrentAgent))
	g.PUT("/api/v1/agents/me", auth(clearsHCCache(handleUpdateCurrentAgent)))
	g.GET("/api/v1/agents/me/teams", auth(handleGetCurrentAgentTeams))
	g.PUT("/api/v1/agents/me/availability", auth(handleUpdateAgentAvailability))
	g.DELETE("/api/v1/agents/me/avatar", auth(clearsHCCache(handleDeleteCurrentAgentAvatar)))

	g.GET("/api/v1/agents/compact", auth(handleGetAgentsCompact))
	g.GET("/api/v1/agents", perm(handleGetAgents, "users:manage"))
	g.GET("/api/v1/agents/{id}", perm(handleGetAgent, "users:manage"))
	g.POST("/api/v1/agents", perm(handleCreateAgent, "users:manage"))
	g.PUT("/api/v1/agents/{id}", perm(clearsHCCache(handleUpdateAgent), "users:manage"))
	g.DELETE("/api/v1/agents/{id}", perm(clearsHCCache(handleDeleteAgent), "users:manage"))
	g.POST("/api/v1/agents/import", perm(clearsHCCache(handleImportAgents), "users:manage"))
	g.GET("/api/v1/agents/import/status", perm(handleGetAgentImportStatus, "users:manage"))
	g.POST("/api/v1/agents/{id}/api-key", perm(handleGenerateAPIKey, "users:manage"))
	g.DELETE("/api/v1/agents/{id}/api-key", perm(handleRevokeAPIKey, "users:manage"))
	g.POST("/api/v1/agents/reset-password", rateLimit(tryAuth(handleResetPassword), "auth"))
	g.POST("/api/v1/agents/set-password", rateLimit(tryAuth(handleSetPassword), "auth"))

	// Contacts.
	g.GET("/api/v1/contacts", perm(handleGetContacts, "contacts:read_all"))
	g.POST("/api/v1/contacts", perm(handleCreateContact, "contacts:write"))
	g.GET("/api/v1/contacts/{id}", perm(handleGetContact, "contacts:read"))
	g.PUT("/api/v1/contacts/{id}", perm(handleUpdateContact, "contacts:write"))
	g.PUT("/api/v1/contacts/{id}/block", perm(handleBlockContact, "contacts:block"))
	g.DELETE("/api/v1/contacts/{id}", perm(handleDeleteContact, "contacts:delete"))
	g.GET("/api/v1/contacts/{id}/export", perm(handleExportContact, "contacts:export"))

	// Contact notes.
	g.GET("/api/v1/contacts/{id}/notes", perm(handleGetContactNotes, "contact_notes:read"))
	g.POST("/api/v1/contacts/{id}/notes", perm(handleCreateContactNote, "contact_notes:write"))
	g.DELETE("/api/v1/contacts/{id}/notes/{note_id}", perm(handleDeleteContactNote, "contact_notes:delete"))

	// Teams.
	g.GET("/api/v1/teams/compact", auth(handleGetTeamsCompact))
	g.GET("/api/v1/teams", perm(handleGetTeams, "teams:manage"))
	g.GET("/api/v1/teams/{id}", perm(handleGetTeam, "teams:manage"))
	g.POST("/api/v1/teams", perm(handleCreateTeam, "teams:manage"))
	g.PUT("/api/v1/teams/{id}", perm(handleUpdateTeam, "teams:manage"))
	g.DELETE("/api/v1/teams/{id}", perm(handleDeleteTeam, "teams:manage"))

	// Automations.
	g.GET("/api/v1/automations/rules", perm(handleGetAutomationRules, "automations:manage"))
	g.GET("/api/v1/automations/rules/{id}", perm(handleGetAutomationRule, "automations:manage"))
	g.POST("/api/v1/automations/rules", perm(handleCreateAutomationRule, "automations:manage"))
	g.PUT("/api/v1/automations/rules/{id}/toggle", perm(handleToggleAutomationRule, "automations:manage"))
	g.PUT("/api/v1/automations/rules/{id}", perm(handleUpdateAutomationRule, "automations:manage"))
	g.PUT("/api/v1/automations/rules/weights", perm(handleUpdateAutomationRuleWeights, "automations:manage"))
	g.PUT("/api/v1/automations/rules/execution-mode", perm(handleUpdateAutomationRuleExecutionMode, "automations:manage"))
	g.DELETE("/api/v1/automations/rules/{id}", perm(handleDeleteAutomationRule, "automations:manage"))

	// Inboxes.
	g.GET("/api/v1/inboxes", auth(handleGetInboxes))
	g.GET("/api/v1/inboxes/{id}", perm(handleGetInbox, "inboxes:manage"))
	g.POST("/api/v1/inboxes", perm(handleCreateInbox, "inboxes:manage"))
	g.PUT("/api/v1/inboxes/{id}/toggle", perm(handleToggleInbox, "inboxes:manage"))
	g.PUT("/api/v1/inboxes/{id}", perm(handleUpdateInbox, "inboxes:manage"))
	g.DELETE("/api/v1/inboxes/{id}", perm(handleDeleteInbox, "inboxes:manage"))

	// OAuth endpoints for email inboxes.
	g.POST("/api/v1/inboxes/oauth/{provider}/authorize", perm(handleOAuthAuthorize, "inboxes:manage"))
	g.GET("/api/v1/inboxes/oauth/{provider}/callback", perm(handleOAuthCallback, "inboxes:manage"))

	// Roles.
	g.GET("/api/v1/roles", auth(handleGetRoles))
	g.GET("/api/v1/roles/{id}", perm(handleGetRole, "roles:manage"))
	g.POST("/api/v1/roles", perm(handleCreateRole, "roles:manage"))
	g.PUT("/api/v1/roles/{id}", perm(handleUpdateRole, "roles:manage"))
	g.DELETE("/api/v1/roles/{id}", perm(handleDeleteRole, "roles:manage"))

	// Webhooks.
	g.GET("/api/v1/webhooks/compact", auth(handleGetWebhooksCompact))
	g.GET("/api/v1/webhooks", perm(handleGetWebhooks, "webhooks:manage"))
	g.GET("/api/v1/webhooks/{id}", perm(handleGetWebhook, "webhooks:manage"))
	g.POST("/api/v1/webhooks", perm(handleCreateWebhook, "webhooks:manage"))
	g.PUT("/api/v1/webhooks/{id}", perm(handleUpdateWebhook, "webhooks:manage"))
	g.DELETE("/api/v1/webhooks/{id}", perm(handleDeleteWebhook, "webhooks:manage"))
	g.PUT("/api/v1/webhooks/{id}/toggle", perm(handleToggleWebhook, "webhooks:manage"))
	g.POST("/api/v1/webhooks/{id}/test", perm(handleTestWebhook, "webhooks:manage"))

	// Context Links.
	g.GET("/api/v1/context-links", perm(handleGetContextLinks, "context_links:manage"))
	g.GET("/api/v1/context-links/active", auth(handleGetActiveContextLinks))
	g.GET("/api/v1/context-links/{id}", perm(handleGetContextLink, "context_links:manage"))
	g.POST("/api/v1/context-links", perm(handleCreateContextLink, "context_links:manage"))
	g.PUT("/api/v1/context-links/{id}", perm(handleUpdateContextLink, "context_links:manage"))
	g.DELETE("/api/v1/context-links/{id}", perm(handleDeleteContextLink, "context_links:manage"))
	g.PUT("/api/v1/context-links/{id}/toggle", perm(handleToggleContextLink, "context_links:manage"))
	g.GET("/api/v1/context-links/{id}/url", auth(handleGetContextLinkURL))

	// Reports.
	g.GET("/api/v1/reports/overview/sla", perm(handleOverviewSLA, "reports:manage"))
	g.GET("/api/v1/reports/overview/counts", perm(handleOverviewCounts, "reports:manage"))
	g.GET("/api/v1/reports/overview/charts", perm(handleOverviewCharts, "reports:manage"))
	g.GET("/api/v1/reports/overview/csat", perm(handleOverviewCSAT, "reports:manage"))
	g.GET("/api/v1/reports/overview/messages", perm(handleOverviewMessageVolume, "reports:manage"))
	g.GET("/api/v1/reports/overview/tags", perm(handleOverviewTagDistribution, "reports:manage"))

	// Templates.
	g.GET("/api/v1/templates", perm(handleGetTemplates, "templates:manage"))
	g.GET("/api/v1/templates/{id}", perm(handleGetTemplate, "templates:manage"))
	g.POST("/api/v1/templates", perm(handleCreateTemplate, "templates:manage"))
	g.PUT("/api/v1/templates/{id}", perm(handleUpdateTemplate, "templates:manage"))
	g.DELETE("/api/v1/templates/{id}", perm(handleDeleteTemplate, "templates:manage"))

	// Business hours.
	g.GET("/api/v1/business-hours", auth(handleGetBusinessHours))
	g.GET("/api/v1/business-hours/{id}", perm(handleGetBusinessHour, "business_hours:manage"))
	g.POST("/api/v1/business-hours", perm(handleCreateBusinessHours, "business_hours:manage"))
	g.PUT("/api/v1/business-hours/{id}", perm(handleUpdateBusinessHours, "business_hours:manage"))
	g.DELETE("/api/v1/business-hours/{id}", perm(handleDeleteBusinessHour, "business_hours:manage"))

	// SLAs.
	g.GET("/api/v1/sla", auth(handleGetSLAs))
	g.GET("/api/v1/sla/{id}", perm(handleGetSLA, "sla:manage"))
	g.POST("/api/v1/sla", perm(handleCreateSLA, "sla:manage"))
	g.PUT("/api/v1/sla/{id}", perm(handleUpdateSLA, "sla:manage"))
	g.DELETE("/api/v1/sla/{id}", perm(handleDeleteSLA, "sla:manage"))

	// [cn-fork] Register AI handlers with feature gate
	registerAIHandlers(g)

	// [cn-fork] Register Help Center handlers with feature gate
	registerHelpCenterHandlers(g, getAndHead)

	// Custom attributes.
	g.GET("/api/v1/custom-attributes", auth(handleGetCustomAttributes))
	g.POST("/api/v1/custom-attributes", perm(handleCreateCustomAttribute, "custom_attributes:manage"))
	g.GET("/api/v1/custom-attributes/{id}", perm(handleGetCustomAttribute, "custom_attributes:manage"))
	g.PUT("/api/v1/custom-attributes/{id}", perm(handleUpdateCustomAttribute, "custom_attributes:manage"))
	g.DELETE("/api/v1/custom-attributes/{id}", perm(handleDeleteCustomAttribute, "custom_attributes:manage"))

	// Actvity logs.
	g.GET("/api/v1/activity-logs", perm(handleGetActivityLogs, "activity_logs:manage"))

	// [cn-fork] Register CSAT handlers with feature gate
	registerCSATHandlers(g)

	// User notifications.
	g.GET("/api/v1/notifications", auth(handleGetUserNotifications))
	g.GET("/api/v1/notifications/stats", auth(handleGetUserNotificationStats))
	g.PUT("/api/v1/notifications/{id}/read", auth(handleMarkNotificationAsRead))
	g.PUT("/api/v1/notifications/read-all", auth(handleMarkAllNotificationsAsRead))
	g.DELETE("/api/v1/notifications/{id}", auth(handleDeleteNotification))
	g.DELETE("/api/v1/notifications", auth(handleDeleteAllNotifications))
	g.GET("/api/v1/notifications/preferences", auth(handleGetNotificationPreferences))
	g.PUT("/api/v1/notifications/preferences", auth(handleUpdateNotificationPreferences))
	g.POST("/api/v1/notifications/push-subscriptions", auth(handleCreatePushSubscription))
	g.DELETE("/api/v1/notifications/push-subscriptions", auth(handleDeletePushSubscription))

	// WebSocket.
	g.GET("/ws", auth(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}))

	// [cn-fork] Register LiveChat handlers with feature gate
	registerLiveChatHandlers(g)

	// Frontend pages.
	getAndHead("/", helpCenterHostHome(notAuthPage(serveIndexPage)))
	g.GET("/inboxes/{all:*}", authPage(serveIndexPage))
	g.GET("/teams/{all:*}", authPage(serveIndexPage))
	g.GET("/views/{all:*}", authPage(serveIndexPage))
	g.GET("/admin/{all:*}", authPage(serveIndexPage))
	g.GET("/contacts/{all:*}", authPage(serveIndexPage))
	g.GET("/reports/{all:*}", authPage(serveIndexPage))
	g.GET("/account/{all:*}", authPage(serveIndexPage))
	g.GET("/reset-password", notAuthPage(serveIndexPage))
	g.GET("/set-password", notAuthPage(serveIndexPage))

	// Assets and static files.
	// FIXME: Reduce the number of routes.
	g.GET("/assets/{all:*}", serveFrontendStaticFiles)
	g.GET("/images/{all:*}", serveFrontendStaticFiles)
	g.GET("/manifest.webmanifest", serveManifest)
	g.GET("/sw.js", serveServiceWorker)
	g.GET("/static/public/{all:*}", serveStaticFiles)

	// Public pages.
	getAndHead("/robots.txt", rateLimit(handleRobotsTxt, "public"))
	getAndHead("/sitemap.xml", rateLimit(handleSitemapIndex, "public"))

	// Health check.
	g.GET("/health", handleHealthCheck)
}

// [cn-fork] registerAIHandlers registers all AI-related routes guarded by feature.AI.
func registerAIHandlers(g *fastglue.Fastglue) {
	// AI completions.
	g.GET("/api/v1/ai/prompts", feat(feature.AI, auth(handleGetAIPrompts)))
	g.GET("/api/v1/ai/prompts/{id}", feat(feature.AI, perm(handleGetAIPrompt, "ai:manage")))
	g.POST("/api/v1/ai/prompts", feat(feature.AI, perm(handleCreateAIPrompt, "ai:manage")))
	g.PUT("/api/v1/ai/prompts/{id}", feat(feature.AI, perm(handleUpdateAIPrompt, "ai:manage")))
	g.DELETE("/api/v1/ai/prompts/{id}", feat(feature.AI, perm(handleDeleteAIPrompt, "ai:manage")))
	g.POST("/api/v1/ai/completion", feat(feature.AI, auth(handleAICompletion)))

	// AI provider config (completion / embedding).
	g.GET("/api/v1/ai/config/{type}", feat(feature.AI, perm(handleGetAIConfig, "ai:manage")))
	g.PUT("/api/v1/ai/config/{type}", feat(feature.AI, perm(handleUpdateAIConfig, "ai:manage")))
	g.POST("/api/v1/ai/config/{type}/test", feat(feature.AI, perm(handleTestAIConfig, "ai:manage")))

	// AI custom tools.
	g.GET("/api/v1/ai/tools", feat(feature.AI, perm(handleGetAITools, "ai:manage")))
	g.GET("/api/v1/ai/tools/{id}", feat(feature.AI, perm(handleGetAITool, "ai:manage")))
	g.POST("/api/v1/ai/tools", feat(feature.AI, perm(handleCreateAITool, "ai:manage")))
	g.PUT("/api/v1/ai/tools/{id}", feat(feature.AI, perm(handleUpdateAITool, "ai:manage")))
	g.DELETE("/api/v1/ai/tools/{id}", feat(feature.AI, perm(handleDeleteAITool, "ai:manage")))

	// AI knowledge base snippets.
	g.GET("/api/v1/ai/snippets", feat(feature.AI, perm(handleGetAISnippets, "ai:manage")))
	g.POST("/api/v1/ai/snippets", feat(feature.AI, perm(handleCreateAISnippet, "ai:manage")))
	g.POST("/api/v1/ai/snippets/import-url", feat(feature.AI, perm(handleImportAISnippetFromURL, "ai:manage")))
	g.PUT("/api/v1/ai/snippets/{id}", feat(feature.AI, perm(handleUpdateAISnippet, "ai:manage")))
	g.DELETE("/api/v1/ai/snippets/{id}", feat(feature.AI, perm(handleDeleteAISnippet, "ai:manage")))

	// AI assistant: reply drafting + copilot chat.
	g.POST("/api/v1/ai/generate-reply", feat(feature.AI, auth(handleAIGenerateReply)))
	g.POST("/api/v1/ai/summarize", feat(feature.AI, perm(handleAISummarizeConversation, "messages:write_private")))
	g.POST("/api/v1/ai/suggest-tags", feat(feature.AI, auth(handleAISuggestTags)))
	g.POST("/api/v1/ai/copilot", feat(feature.AI, auth(handleAICopilot)))
	g.GET("/api/v1/ai/copilot/messages", feat(feature.AI, auth(handleGetCopilotMessages)))
	g.DELETE("/api/v1/ai/copilot/messages", feat(feature.AI, auth(handleClearCopilotMessages)))

	// Autonomous AI agents (assistants).
	g.GET("/api/v1/ai/assistants/compact", feat(feature.AI, auth(handleGetAIAssistantsCompact)))
	g.GET("/api/v1/ai/assistants", feat(feature.AI, perm(handleGetAIAssistants, "ai:manage")))
	g.GET("/api/v1/ai/assistants/{id}", feat(feature.AI, perm(handleGetAIAssistant, "ai:manage")))
	g.POST("/api/v1/ai/assistants", feat(feature.AI, perm(handleCreateAIAssistant, "ai:manage")))
	g.PUT("/api/v1/ai/assistants/{id}", feat(feature.AI, perm(clearsHCCache(handleUpdateAIAssistant), "ai:manage")))
	g.DELETE("/api/v1/ai/assistants/{id}", feat(feature.AI, perm(clearsHCCache(handleDeleteAIAssistant), "ai:manage")))
	g.POST("/api/v1/ai/assistants/{id}/preview", feat(feature.AI, perm(handleAIAssistantPreview, "ai:manage")))
	g.GET("/api/v1/ai/assistants/{id}/stats", feat(feature.AI, perm(handleGetAIAssistantStats, "ai:manage")))

	// AI FAQ learning: review queue for suggestions mined from resolved conversations + on/off setting.
	g.GET("/api/v1/ai/faq-suggestions", feat(feature.AI, perm(handleGetAIFaqSuggestions, "ai:manage")))
	g.POST("/api/v1/ai/faq-suggestions/{id}/approve", feat(feature.AI, perm(handleApproveAIFaqSuggestion, "ai:manage")))
	g.POST("/api/v1/ai/faq-suggestions/{id}/reject", feat(feature.AI, perm(handleRejectAIFaqSuggestion, "ai:manage")))
	g.GET("/api/v1/ai/faq-learning", feat(feature.AI, perm(handleGetAIFaqLearning, "ai:manage")))
	g.PUT("/api/v1/ai/faq-learning", feat(feature.AI, perm(handleUpdateAIFaqLearning, "ai:manage")))
}

// [cn-fork] registerHelpCenterHandlers registers all Help Center routes guarded by feature.HelpCenter.
func registerHelpCenterHandlers(g *fastglue.Fastglue, getAndHead func(path string, h fastglue.FastRequestHandler)) {
	// Help centers.
	g.GET("/api/v1/help-centers", feat(feature.HelpCenter, perm(handleGetHelpCenters, "help_center:manage")))
	g.GET("/api/v1/help-centers/locales", feat(feature.HelpCenter, perm(handleGetHelpCenterLocales, "help_center:manage")))
	g.GET("/api/v1/help-centers/{id}", feat(feature.HelpCenter, perm(handleGetHelpCenter, "help_center:manage")))
	g.GET("/api/v1/help-centers/{id}/tree", feat(feature.HelpCenter, perm(handleGetHelpCenterTree, "help_center:manage")))
	g.POST("/api/v1/help-centers", feat(feature.HelpCenter, perm(clearsHCCache(handleCreateHelpCenter), "help_center:manage")))
	g.PUT("/api/v1/help-centers/{id}", feat(feature.HelpCenter, perm(clearsHCCache(handleUpdateHelpCenter), "help_center:manage")))
	g.POST("/api/v1/help-centers/{id}/preview", feat(feature.HelpCenter, perm(handleHelpCenterPreview, "help_center:manage")))
	g.PUT("/api/v1/help-centers/{id}/toggle", feat(feature.HelpCenter, perm(clearsHCCache(handleToggleHelpCenterActive), "help_center:manage")))
	g.DELETE("/api/v1/help-centers/{id}", feat(feature.HelpCenter, perm(clearsHCCache(handleDeleteHelpCenter), "help_center:manage")))
	g.GET("/api/v1/help-centers/{hc_id}/collections", feat(feature.HelpCenter, perm(handleGetCollections, "help_center:manage")))
	g.POST("/api/v1/help-centers/{hc_id}/collections", feat(feature.HelpCenter, perm(clearsHCCache(handleCreateCollection), "help_center:manage")))
	g.PUT("/api/v1/help-centers/{hc_id}/collections/{id}", feat(feature.HelpCenter, perm(clearsHCCache(handleUpdateCollection), "help_center:manage")))
	g.DELETE("/api/v1/help-centers/{hc_id}/collections/{id}", feat(feature.HelpCenter, perm(clearsHCCache(handleDeleteCollection), "help_center:manage")))
	g.PUT("/api/v1/help-centers/{hc_id}/collection-sort-order", feat(feature.HelpCenter, perm(clearsHCCache(handleUpdateCollectionSortOrders), "help_center:manage")))
	g.PUT("/api/v1/collections/{id}/toggle", feat(feature.HelpCenter, perm(clearsHCCache(handleToggleCollection), "help_center:manage")))
	g.PUT("/api/v1/collections/{col_id}/article-sort-order", feat(feature.HelpCenter, perm(clearsHCCache(handleUpdateArticleSortOrders), "help_center:manage")))
	g.GET("/api/v1/collections/{col_id}/articles/{id}", feat(feature.HelpCenter, perm(handleGetArticle, "help_center:manage")))
	g.POST("/api/v1/collections/{col_id}/articles", feat(feature.HelpCenter, perm(clearsHCCache(handleCreateArticle), "help_center:manage")))
	g.DELETE("/api/v1/collections/{col_id}/articles/{id}", feat(feature.HelpCenter, perm(clearsHCCache(handleDeleteArticle), "help_center:manage")))
	g.PUT("/api/v1/articles/{id}", feat(feature.HelpCenter, perm(clearsHCCache(handleUpdateArticle), "help_center:manage")))
	g.PUT("/api/v1/articles/{id}/collection", feat(feature.HelpCenter, perm(clearsHCCache(handleMoveArticle), "help_center:manage")))
	g.PUT("/api/v1/articles/{id}/status", feat(feature.HelpCenter, perm(clearsHCCache(handleUpdateArticleStatus), "help_center:manage")))
	g.GET("/api/v1/help-centers/{id}/insights", feat(feature.HelpCenter, perm(handleGetHelpCenterInsights, "help_center:manage")))

	// Public help center JSON API.
	g.GET("/api/v1/public/help-centers/{slug}/tree", feat(feature.HelpCenter, rateLimit(handleGetPublicHelpCenterTree, "public")))
	g.GET("/api/v1/public/help-centers/{slug}/articles/{article_slug}", feat(feature.HelpCenter, rateLimit(handleGetPublicHelpCenterArticle, "public")))
	g.GET("/api/v1/public/help-centers/{slug}/search", feat(feature.HelpCenter, rateLimit(handlePublicHelpCenterSearch, "public")))
	g.POST("/api/v1/public/help-centers/{slug}/articles/{article_slug}/feedback", feat(feature.HelpCenter, rateLimit(handleHelpCenterArticleFeedback, "public")))

	// Public help center web pages.
	getAndHead("/hc/{slug}", feat(feature.HelpCenter, rateLimit(handleRedirectHelpCenterHome, "public")))
	getAndHead("/hc/{slug}/{locale}", feat(feature.HelpCenter, rateLimit(cachedHCPage(handleShowHelpCenterHome), "public")))
	getAndHead("/hc/{slug}/{locale}/sitemap.xml", feat(feature.HelpCenter, rateLimit(handleHelpCenterSitemap, "public")))
	getAndHead("/hc/{slug}/{locale}/search", feat(feature.HelpCenter, rateLimit(cachedHCNoIndexPage(handleHelpCenterSearch), "public")))
	getAndHead("/hc/{slug}/{locale}/collections/{collection_slug}", feat(feature.HelpCenter, rateLimit(cachedHCPage(handleShowHelpCenterCollection), "public")))
	getAndHead("/hc/{slug}/{locale}/articles/{article_slug}", feat(feature.HelpCenter, rateLimit(countArticleView(cachedHCPage(handleShowHelpCenterArticle)), "public")))
}

// [cn-fork] registerCSATHandlers registers CSAT routes guarded by feature.CSAT.
func registerCSATHandlers(g *fastglue.Fastglue) {
	g.POST("/api/v1/csat/{uuid}/response", feat(feature.CSAT, rateLimit(handleSubmitCSATResponse, "public")))
	g.GET("/csat/{uuid}", feat(feature.CSAT, rateLimit(handleShowCSAT, "public")))
	g.GET("/csat/{uuid}/widget", feat(feature.CSAT, rateLimit(handleShowCSATWidget, "public")))
	g.POST("/csat/{uuid}", feat(feature.CSAT, rateLimit(handleUpdateCSATResponse, "public")))
}

// [cn-fork] registerLiveChatHandlers registers all LiveChat / Widget routes guarded by feature.LiveChat.
func registerLiveChatHandlers(g *fastglue.Fastglue) {
	// Live chat widget websocket.
	g.GET("/widget/ws", feat(feature.LiveChat, rateLimit(handleWidgetWS, "widget")))

	// Widget APIs.
	g.GET("/api/v1/widget/chat/settings/launcher", feat(feature.LiveChat, rateLimit(validateWidgetInbox(handleGetChatLauncherSettings), "widget")))
	g.GET("/api/v1/widget/chat/settings", feat(feature.LiveChat, rateLimit(validateWidgetInbox(handleGetChatSettings), "widget")))
	g.POST("/api/v1/widget/chat/auth/exchange", feat(feature.LiveChat, rateLimit(validateWidgetInbox(handleAuthExchange), "widget")))
	g.GET("/api/v1/widget/chat/auth/me", feat(feature.LiveChat, rateLimit(widgetAuth(handleWidgetAuthMe), "widget")))
	g.POST("/api/v1/widget/chat/conversations/init", feat(feature.LiveChat, rateLimit(widgetAuth(handleChatInit), "widget")))
	g.GET("/api/v1/widget/chat/conversations", feat(feature.LiveChat, rateLimit(widgetAuth(handleGetConversations), "widget")))
	g.POST("/api/v1/widget/chat/conversations/{uuid}/update-last-seen", feat(feature.LiveChat, rateLimit(widgetAuth(handleChatUpdateLastSeen), "widget")))
	g.GET("/api/v1/widget/chat/conversations/{uuid}", feat(feature.LiveChat, rateLimit(widgetAuth(handleChatGetConversation), "widget")))
	g.POST("/api/v1/widget/chat/conversations/{uuid}/message", feat(feature.LiveChat, rateLimit(widgetAuth(handleChatSendMessage), "widget")))
	g.POST("/api/v1/widget/media/upload", feat(feature.LiveChat, rateLimit(widgetAuth(handleWidgetMediaUpload), "widget")))

	// Widget page and assets.
	g.GET("/widget", feat(feature.LiveChat, validateWidgetInbox(serveWidgetIndexPage)))
	g.GET("/widget.js", feat(feature.LiveChat, serveWidgetJS))
	g.GET("/widget/assets/{all:*}", feat(feature.LiveChat, serveWidgetStaticFiles))
}

// serveIndexPage serves the main index page of the application.
func serveIndexPage(r *fastglue.Request) error {
	app := r.Context.(*App)

	// Prevent caching of the index page.
	r.RequestCtx.Response.Header.Add("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	r.RequestCtx.Response.Header.Add("Pragma", "no-cache")
	r.RequestCtx.Response.Header.Add("Expires", "-1")

	// Serve the index.html file from the embedded filesystem.
	file, err := app.fs.Get(path.Join(frontendDir, "index.html"))
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "text/html")
	r.RequestCtx.Response.SetBodyRaw(file.ReadBytes())

	// Set CSRF cookie if not already set.
	if err := app.auth.SetCSRFCookie(r); err != nil {
		app.lo.Error("error setting csrf cookie", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	return nil
}

// serveWidgetIndexPage serves the widget index page of the application.
func serveWidgetIndexPage(r *fastglue.Request) error {
	app := r.Context.(*App)

	// Prevent caching of the index page.
	r.RequestCtx.Response.Header.Add("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	r.RequestCtx.Response.Header.Add("Pragma", "no-cache")
	r.RequestCtx.Response.Header.Add("Expires", "-1")

	// CSP headers if trusted domains is set.
	if config, err := getWidgetConfig(r); err == nil && len(config.TrustedDomains) > 0 {
		csp := "frame-ancestors 'self' " + strings.Join(config.TrustedDomains, " ")
		r.RequestCtx.Response.Header.Set("Content-Security-Policy", csp)
	}

	// Serve the index.html file from the embedded filesystem.
	file, err := app.fs.Get(path.Join(widgetDir, "index.html"))
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "text/html")
	r.RequestCtx.Response.SetBodyRaw(file.ReadBytes())

	return nil
}

// serveStaticFiles serves static assets from the filesystem.
func serveStaticFiles(r *fastglue.Request) error {
	app := r.Context.(*App)

	filePath := string(r.RequestCtx.Path())

	file, err := app.fs.Get(filePath)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}

	body := file.ReadBytes()
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.Response.Header.Set("Cache-Control", "public, max-age=86400")
	r.RequestCtx.Response.SetBodyRaw(body)
	return nil
}

// serveFrontendStaticFiles serves static assets from the embedded filesystem.
func serveFrontendStaticFiles(r *fastglue.Request) error {
	app := r.Context.(*App)

	// Get the requested file path.
	filePath := string(r.RequestCtx.Path())

	// Fetch and serve the file from the embedded filesystem.
	finalPath := filepath.Join(frontendDir, filePath)
	file, err := app.fs.Get(finalPath)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}

	body := file.ReadBytes()
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.Response.SetBodyRaw(body)
	return nil
}

func serveManifest(r *fastglue.Request) error {
	return serveMainFrontendFile(r, "manifest.webmanifest", "application/manifest+json", "no-cache")
}

func serveServiceWorker(r *fastglue.Request) error {
	r.RequestCtx.Response.Header.Set("Service-Worker-Allowed", "/")
	return serveMainFrontendFile(r, "sw.js", "application/javascript", "no-cache")
}

func serveMainFrontendFile(r *fastglue.Request, name, contentType, cacheControl string) error {
	app := r.Context.(*App)
	file, err := app.fs.Get(filepath.Join(frontendDir, name))
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.Response.Header.Set("Cache-Control", cacheControl)
	r.RequestCtx.SetBody(file.ReadBytes())
	return nil
}

// serveWidgetStaticFiles serves widget static assets from the embedded filesystem.
func serveWidgetStaticFiles(r *fastglue.Request) error {
	app := r.Context.(*App)

	filePath := string(r.RequestCtx.Path())
	finalPath := filepath.Join(widgetDir, strings.TrimPrefix(filePath, "/widget"))

	file, err := app.fs.Get(finalPath)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}

	body := file.ReadBytes()
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.Response.SetBodyRaw(body)
	return nil
}

// serveWidgetJS serves the widget JavaScript file.
func serveWidgetJS(r *fastglue.Request) error {
	app := r.Context.(*App)

	r.RequestCtx.Response.Header.Set("Content-Type", "application/javascript")
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-cache")

	file, err := app.fs.Get("static/widget.js")
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}

	r.RequestCtx.Response.SetBodyRaw(file.ReadBytes())
	return nil
}

// getIDsParam parses a comma separated list of positive IDs from a query param.
func getIDsParam(r *fastglue.Request, name string) []int {
	raw := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek(name)))
	if raw == "" {
		return nil
	}
	var ids []int
	for part := range strings.SplitSeq(raw, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
		if len(ids) == maxIDsParam {
			break
		}
	}
	return ids
}

// getPagination extracts page and page_size from query params with defaults.
func getPagination(r *fastglue.Request) (page, pageSize int) {
	page, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
	pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 30
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// getOptionalPagination reads page/page_size, returning pageSize 0 (fetch everything) when absent.
func getOptionalPagination(r *fastglue.Request) (page, pageSize int) {
	page, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
	pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
	if page < 1 {
		page = 1
	}
	if pageSize < 0 {
		pageSize = 0
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// sendErrorEnvelope sends a standardized error response to the client.
func sendErrorEnvelope(r *fastglue.Request, err error) error {
	e, ok := err.(envelope.Error)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Error interface conversion failed", nil, fastglue.ErrorType(envelope.GeneralError))
	}
	return r.SendErrorEnvelope(e.Code, e.Error(), e.Data, fastglue.ErrorType(e.ErrorType))
}

// handleHealthCheck handles the health check endpoint.
func handleHealthCheck(r *fastglue.Request) error {
	return r.SendEnvelope(true)
}

// [cn-fork]
package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/abhinavxd/libredesk/internal/feature"
	"github.com/knadh/go-i18n"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestFeatMiddleware(t *testing.T) {
	featureMgr, err := feature.New(nil, nil)
	if err != nil {
		t.Fatalf("failed to create feature manager: %v", err)
	}

	i18nInst, err := i18n.New([]byte(`{
		"_.code": "en-US",
		"_.name": "English",
		"globals.messages.featureDisabled": "This feature is currently disabled"
	}`))
	if err != nil {
		t.Fatalf("failed to load i18n: %v", err)
	}

	app := &App{
		feature: featureMgr,
		i18n:    i18nInst,
	}

	dummyHandlerCalled := false
	dummyHandler := func(r *fastglue.Request) error {
		dummyHandlerCalled = true
		return r.SendEnvelope("ok")
	}

	// AI is disabled by default
	aiGuardedHandler := feat(feature.AI, dummyHandler)

	ctx := &fasthttp.RequestCtx{}
	req := &fastglue.Request{RequestCtx: ctx, Context: app}

	// 1. When disabled -> 404
	err = aiGuardedHandler(req)
	if err != nil {
		t.Fatalf("unexpected error executing handler: %v", err)
	}
	if ctx.Response.StatusCode() != http.StatusNotFound {
		t.Errorf("expected 404 for disabled feature, got %d", ctx.Response.StatusCode())
	}
	if dummyHandlerCalled {
		t.Errorf("dummyHandler should not be called when feature is disabled")
	}

	var resp map[string]any
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("error unmarshalling response body: %v", err)
	}
	if resp["message"] != "This feature is currently disabled" {
		t.Errorf("expected message 'This feature is currently disabled', got %v", resp["message"])
	}

	// 2. Enable AI -> should pass through
	featureMgr.Set(feature.AI, true)
	ctx.Response.Reset()
	dummyHandlerCalled = false

	err = aiGuardedHandler(req)
	if err != nil {
		t.Fatalf("unexpected error executing handler: %v", err)
	}
	if !dummyHandlerCalled {
		t.Errorf("dummyHandler should be called when feature is enabled")
	}
	if ctx.Response.StatusCode() != http.StatusOK {
		t.Errorf("expected 200 for enabled feature, got %d", ctx.Response.StatusCode())
	}
}

func TestRegisteredFeatureRoutesGated(t *testing.T) {
	featureMgr, err := feature.New(nil, nil)
	if err != nil {
		t.Fatalf("failed to create feature manager: %v", err)
	}

	i18nInst, err := i18n.New([]byte(`{
		"_.code": "en-US",
		"_.name": "English",
		"globals.messages.featureDisabled": "This feature is currently disabled"
	}`))
	if err != nil {
		t.Fatalf("failed to load i18n: %v", err)
	}

	app := &App{
		feature: featureMgr,
		i18n:    i18nInst,
	}

	g := fastglue.NewGlue()
	g.SetContext(app)
	registerAIHandlers(g)
	registerLiveChatHandlers(g)
	registerCSATHandlers(g)

	// Ensure all gated features are disabled for test
	featureMgr.Set(feature.AI, false)
	featureMgr.Set(feature.LiveChat, false)
	featureMgr.Set(feature.CSAT, false)

	testCases := []struct {
		name   string
		method string
		uri    string
	}{
		{"AI Prompts", http.MethodGet, "/api/v1/ai/prompts"},
		{"LiveChat Settings", http.MethodGet, "/api/v1/widget/chat/settings"},
		{"CSAT Response", http.MethodPost, "/api/v1/csat/test-uuid/response"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.Header.SetMethod(tc.method)
			ctx.Request.SetRequestURI(tc.uri)
			g.Router.Handler(ctx)

			if ctx.Response.StatusCode() != http.StatusNotFound {
				t.Errorf("expected 404 for %s, got %d", tc.uri, ctx.Response.StatusCode())
			}

			var resp map[string]any
			if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
				t.Fatalf("error unmarshalling response for %s: %v", tc.uri, err)
			}
			if resp["message"] != "This feature is currently disabled" {
				t.Errorf("expected featureDisabled message for %s, got %v", tc.uri, resp["message"])
			}
		})
	}
}

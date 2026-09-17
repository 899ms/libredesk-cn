// [cn-fork]
package main

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/knadh/go-i18n"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestHandleAuthExchangeValidation(t *testing.T) {
	lo := logf.New(logf.Opts{Writer: io.Discard})
	app := &App{
		i18n: testutil.NewI18n(t),
		lo:   &lo,
	}

	secret := "test-secret-key-12345"
	inbox := imodels.Inbox{
		ID:     1,
		Secret: null.StringFrom(secret),
	}
	cfg := livechat.Config{}

	generateJWT := func(c Claims) string {
		c.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
		tokenStr, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}
		return tokenStr
	}

	tests := []struct {
		name              string
		claims            Claims
		wantStatus        int
		wantErrorContains string
	}{
		{
			name: "missing external_user_id",
			claims: Claims{
				Email:       "user@example.com",
				FirstName:   "Alice",
				PhoneNumber: "13800000000",
			},
			wantStatus:        fasthttp.StatusBadRequest,
			wantErrorContains: "external_user_id",
		},
		{
			name: "missing both email and phone_number",
			claims: Claims{
				ExternalUserID: "ext-1",
				FirstName:      "Alice",
			},
			wantStatus:        fasthttp.StatusBadRequest,
			wantErrorContains: "Either email or phone number is required",
		},
		{
			name: "only phone_number passes validation",
			claims: Claims{
				ExternalUserID: "ext-2",
				FirstName:      "Bob",
				PhoneNumber:    "13900000000",
			},
			// Validation passes, fails downstream because DB/user store is not initialized
			wantStatus:        fasthttp.StatusInternalServerError,
			wantErrorContains: "Something went wrong",
		},
		{
			name: "only email passes validation",
			claims: Claims{
				ExternalUserID: "ext-3",
				FirstName:      "Charlie",
				Email:          "charlie@example.com",
			},
			// Validation passes, fails downstream because DB/user store is not initialized
			wantStatus:        fasthttp.StatusInternalServerError,
			wantErrorContains: "Something went wrong",
		},
		{
			name: "both email and phone_number pass validation",
			claims: Claims{
				ExternalUserID: "ext-4",
				FirstName:      "David",
				Email:          "david@example.com",
				PhoneNumber:    "13700000000",
			},
			wantStatus:        fasthttp.StatusInternalServerError,
			wantErrorContains: "Something went wrong",
		},
		{
			name: "phone_number too long",
			claims: Claims{
				ExternalUserID: "ext-5",
				FirstName:      "Eve",
				PhoneNumber:    strings.Repeat("1", 30),
			},
			wantStatus:        fasthttp.StatusBadRequest,
			wantErrorContains: "Phone number",
		},
		{
			name: "email too long",
			claims: Claims{
				ExternalUserID: "ext-6",
				FirstName:      "Frank",
				Email:          strings.Repeat("a", 300) + "@example.com",
			},
			wantStatus:        fasthttp.StatusBadRequest,
			wantErrorContains: "Email",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token := generateJWT(tc.claims)
			bodyBytes, _ := json.Marshal(map[string]string{"jwt": token})

			ctx := &fasthttp.RequestCtx{}
			ctx.Request.Header.SetMethod("POST")
			ctx.Request.Header.SetContentType("application/json")
			ctx.Request.SetBody(bodyBytes)
			ctx.SetUserValue(ctxWidgetInbox, inbox)
			ctx.SetUserValue(ctxWidgetConfig, cfg)

			req := &fastglue.Request{RequestCtx: ctx, Context: app}

			err := handleAuthExchange(req)
			if err != nil {
				t.Fatalf("unexpected handler error: %v", err)
			}

			if ctx.Response.StatusCode() != tc.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tc.wantStatus, ctx.Response.StatusCode(), string(ctx.Response.Body()))
			}

			var resp map[string]any
			if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
				t.Fatalf("failed to parse response JSON: %v", err)
			}
			msg, _ := resp["message"].(string)
			if tc.wantErrorContains != "" && !strings.Contains(msg, tc.wantErrorContains) {
				t.Errorf("expected error message to contain %q, got %q", tc.wantErrorContains, msg)
			}
		})
	}
}

func TestHandleAuthExchangeChineseMessage(t *testing.T) {
	lo := logf.New(logf.Opts{Writer: io.Discard})
	zhI18n, err := i18n.NewFromFile(filepath.Join("..", "i18n", "zh-CN.json"))
	if err != nil {
		t.Fatalf("loading zh-CN i18n: %v", err)
	}

	app := &App{
		i18n: zhI18n,
		lo:   &lo,
	}

	secret := "test-secret"
	inbox := imodels.Inbox{
		ID:     1,
		Secret: null.StringFrom(secret),
	}
	cfg := livechat.Config{}

	claims := Claims{
		ExternalUserID: "ext-1",
		FirstName:      "测试",
	}
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}

	bodyBytes, _ := json.Marshal(map[string]string{"jwt": tokenStr})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody(bodyBytes)
	ctx.SetUserValue(ctxWidgetInbox, inbox)
	ctx.SetUserValue(ctxWidgetConfig, cfg)

	req := &fastglue.Request{RequestCtx: ctx, Context: app}

	if err := handleAuthExchange(req); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("expected 400, got %d", ctx.Response.StatusCode())
	}

	var resp map[string]any
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["message"] != "必须提供电子邮件或手机号。" {
		t.Errorf("expected '必须提供电子邮件或手机号。', got %q", resp["message"])
	}
}


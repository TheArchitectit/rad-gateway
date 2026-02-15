package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

type ctxKey string

const (
	KeyRequestID ctxKey = "request_id"
	KeyTraceID   ctxKey = "trace_id"
	KeyAPIKey    ctxKey = "api_key"
	KeyAPIName   ctxKey = "api_key_name"
)

type Authenticator struct {
	keys map[string]string
}

func NewAuthenticator(keys map[string]string) *Authenticator {
	copyMap := map[string]string{}
	for k, v := range keys {
		copyMap[k] = v
	}
	return &Authenticator{keys: copyMap}
}

func (a *Authenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := extractAPIKey(r)
		if secret == "" {
			http.Error(w, `{"error":{"message":"missing api key","code":401}}`, http.StatusUnauthorized)
			return
		}

		name := ""
		for k, v := range a.keys {
			if v == secret {
				name = k
				break
			}
		}
		if name == "" {
			http.Error(w, `{"error":{"message":"invalid api key","code":401}}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), KeyAPIKey, secret)
		ctx = context.WithValue(ctx, KeyAPIName, name)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WithRequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = newID()
		}
		traceID := r.Header.Get("AH-Trace-Id")
		if traceID == "" {
			traceID = r.Header.Get("X-Trace-Id")
		}
		if traceID == "" {
			traceID = reqID
		}

		w.Header().Set("X-Request-Id", reqID)
		w.Header().Set("X-Trace-Id", traceID)

		ctx := context.WithValue(r.Context(), KeyRequestID, reqID)
		ctx = context.WithValue(ctx, KeyTraceID, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) string {
	v, _ := ctx.Value(KeyRequestID).(string)
	return v
}

func GetTraceID(ctx context.Context) string {
	v, _ := ctx.Value(KeyTraceID).(string)
	return v
}

func GetAPIKeyName(ctx context.Context) string {
	v, _ := ctx.Value(KeyAPIName).(string)
	return v
}

func extractAPIKey(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	if k := strings.TrimSpace(r.Header.Get("x-api-key")); k != "" {
		return k
	}
	if k := strings.TrimSpace(r.Header.Get("x-goog-api-key")); k != "" {
		return k
	}
	return strings.TrimSpace(r.URL.Query().Get("key"))
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

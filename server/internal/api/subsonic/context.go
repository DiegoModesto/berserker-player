package subsonic

import (
	"context"
	"net/http"
	"time"
)

func withUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxUserID, id)
}

func userID(r *http.Request) string {
	if v, ok := r.Context().Value(ctxUserID).(string); ok {
		return v
	}
	return ""
}

func nowUTC() time.Time { return time.Now().UTC() }

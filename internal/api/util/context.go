package util

import (
	"context"

	"github.com/google/uuid"
)

type contextKey int

const (
	_ contextKey = iota
	userKey
	resourceIDKey
	requestIDKey
)

func ContextWithUser(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userKey, userID)
}

func UserFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userKey).(uuid.UUID)
	return userID, ok
}

func ContextWithResourceID(ctx context.Context, resourceID any) context.Context {
	return context.WithValue(ctx, resourceIDKey, resourceID)
}

func ResourceIDFromContext(ctx context.Context) (any, bool) {
	resourceID := ctx.Value(resourceIDKey)
	if resourceID == nil {
		return nil, false
	}
	return resourceID, true
}

func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey).(string)
	return requestID, ok
}

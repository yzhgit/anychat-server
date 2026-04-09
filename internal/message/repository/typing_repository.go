package repository

import (
	"context"
	"fmt"
	"time"

	pkgredis "github.com/anychat/server/pkg/redis"
)

// TypingRepository typing status cache repository
type TypingRepository interface {
	SetState(ctx context.Context, conversationID, fromUserID string, ttl time.Duration) error
	ClearState(ctx context.Context, conversationID, fromUserID string) error
	AcquireEmitToken(ctx context.Context, conversationID, fromUserID string, ttl time.Duration) (bool, error)
}

type typingRepositoryImpl struct {
	cache *pkgredis.Client
}

// NewTypingRepository creates typing status cache repository
func NewTypingRepository(cache *pkgredis.Client) TypingRepository {
	return &typingRepositoryImpl{cache: cache}
}

func (r *typingRepositoryImpl) SetState(ctx context.Context, conversationID, fromUserID string, ttl time.Duration) error {
	return r.cache.Set(ctx, r.stateKey(conversationID, fromUserID), "1", ttl)
}

func (r *typingRepositoryImpl) ClearState(ctx context.Context, conversationID, fromUserID string) error {
	return r.cache.Del(ctx, r.stateKey(conversationID, fromUserID), r.emitKey(conversationID, fromUserID))
}

func (r *typingRepositoryImpl) AcquireEmitToken(ctx context.Context, conversationID, fromUserID string, ttl time.Duration) (bool, error) {
	return r.cache.GetClient().SetNX(ctx, r.emitKey(conversationID, fromUserID), "1", ttl).Result()
}

func (r *typingRepositoryImpl) stateKey(conversationID, fromUserID string) string {
	return fmt.Sprintf("msg:typing:state:%s:%s", conversationID, fromUserID)
}

func (r *typingRepositoryImpl) emitKey(conversationID, fromUserID string) string {
	return fmt.Sprintf("msg:typing:emit:%s:%s", conversationID, fromUserID)
}

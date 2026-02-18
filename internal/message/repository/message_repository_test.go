package repository

import (
	"context"
	"testing"

	"github.com/anychat/server/internal/message/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(&model.Message{}, &model.MessageReadReceipt{}, &model.ConversationSequence{})
	require.NoError(t, err)

	return db
}

func TestMessageRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	msg := &model.Message{
		MessageID:        "msg-123",
		ConversationID:   "conv-123",
		ConversationType: model.ConversationTypeSingle,
		SenderID:         "user-1",
		ContentType:      model.ContentTypeText,
		Content:          `{"text":"Hello"}`,
		Sequence:         1,
		Status:           model.MessageStatusNormal,
	}

	err := repo.Create(ctx, msg)
	assert.NoError(t, err)
	assert.NotZero(t, msg.ID)
}

func TestMessageRepository_GetByMessageID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	// 创建测试消息
	messageID := "msg-test-123"
	msg := &model.Message{
		MessageID:        messageID,
		ConversationID:   "conv-123",
		ConversationType: model.ConversationTypeSingle,
		SenderID:         "user-1",
		ContentType:      model.ContentTypeText,
		Content:          `{"text":"Test"}`,
		Sequence:         1,
		Status:           model.MessageStatusNormal,
	}
	err := repo.Create(ctx, msg)
	require.NoError(t, err)

	// 测试获取消息
	got, err := repo.GetByMessageID(ctx, messageID)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, messageID, got.MessageID)
	assert.Equal(t, "conv-123", got.ConversationID)
}

func TestMessageRepository_GetByConversation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	conversationID := "conv-123"

	// 创建多条消息
	for i := 1; i <= 5; i++ {
		msg := &model.Message{
			MessageID:        string(rune(i)),
			ConversationID:   conversationID,
			ConversationType: model.ConversationTypeSingle,
			SenderID:         "user-1",
			ContentType:      model.ContentTypeText,
			Content:          `{"text":"Test"}`,
			Sequence:         int64(i),
			Status:           model.MessageStatusNormal,
		}
		err := repo.Create(ctx, msg)
		require.NoError(t, err)
	}

	// 测试获取消息
	messages, err := repo.GetByConversation(ctx, conversationID, 1, 5, 10, false)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(messages))

	// 验证排序（正序）
	if len(messages) > 1 {
		assert.Less(t, messages[0].Sequence, messages[1].Sequence)
	}

	// 测试倒序
	messagesReverse, err := repo.GetByConversation(ctx, conversationID, 1, 5, 10, true)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(messagesReverse))
	if len(messagesReverse) > 1 {
		assert.Greater(t, messagesReverse[0].Sequence, messagesReverse[1].Sequence)
	}
}

func TestMessageRepository_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	// 创建测试消息
	messageID := "msg-to-recall"
	msg := &model.Message{
		MessageID:        messageID,
		ConversationID:   "conv-123",
		ConversationType: model.ConversationTypeSingle,
		SenderID:         "user-1",
		ContentType:      model.ContentTypeText,
		Content:          `{"text":"Test"}`,
		Sequence:         1,
		Status:           model.MessageStatusNormal,
	}
	err := repo.Create(ctx, msg)
	require.NoError(t, err)

	// 更新状态为已撤回
	err = repo.UpdateStatus(ctx, messageID, model.MessageStatusRecall)
	assert.NoError(t, err)

	// 验证状态已更新
	got, err := repo.GetByMessageID(ctx, messageID)
	require.NoError(t, err)
	assert.Equal(t, int16(model.MessageStatusRecall), got.Status)
}

func TestMessageRepository_CountByConversation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	conversationID := "conv-test-count"

	// 创建3条消息
	for i := 1; i <= 3; i++ {
		msg := &model.Message{
			MessageID:        string(rune(i + 100)),
			ConversationID:   conversationID,
			ConversationType: model.ConversationTypeSingle,
			SenderID:         "user-1",
			ContentType:      model.ContentTypeText,
			Content:          `{"text":"Test"}`,
			Sequence:         int64(i),
			Status:           model.MessageStatusNormal,
		}
		err := repo.Create(ctx, msg)
		require.NoError(t, err)
	}

	count, err := repo.CountByConversation(ctx, conversationID)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestSequenceRepository_IncrementAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	conversationID := "conv-seq-test"

	// 第一次调用应该返回1
	seq1, err := repo.IncrementAndGet(ctx, conversationID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), seq1)

	// 第二次调用应该返回2
	seq2, err := repo.IncrementAndGet(ctx, conversationID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), seq2)

	// 第三次调用应该返回3
	seq3, err := repo.IncrementAndGet(ctx, conversationID)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), seq3)
}

func TestSequenceRepository_GetCurrentSeq(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSequenceRepository(db)
	ctx := context.Background()

	conversationID := "conv-current-seq"

	// 先增加几次
	repo.IncrementAndGet(ctx, conversationID)
	repo.IncrementAndGet(ctx, conversationID)
	repo.IncrementAndGet(ctx, conversationID)

	// 获取当前序列号
	seq, err := repo.GetCurrentSeq(ctx, conversationID)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), seq)
}

func TestReadReceiptRepository_Upsert(t *testing.T) {
	t.Skip("Upsert uses PostgreSQL-specific ON CONFLICT syntax not supported by SQLite. Test with PostgreSQL instead.")

	// This test requires PostgreSQL due to ON CONFLICT syntax
	// For integration tests, use the actual PostgreSQL database
}

func TestReadReceiptRepository_GetByConversation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReadReceiptRepository(db)
	ctx := context.Background()

	conversationID := "conv-group-123"

	// 创建多个用户的已读回执
	for i := 1; i <= 3; i++ {
		receipt := &model.MessageReadReceipt{
			ConversationID:    conversationID,
			UserID:            string(rune(i)),
			LastReadSeq:       int64(i * 10),
			LastReadMessageID: nil,
		}
		err := repo.Upsert(ctx, receipt)
		require.NoError(t, err)
	}

	// 获取会话的所有已读回执
	receipts, err := repo.GetByConversation(ctx, conversationID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(receipts))
}

// 辅助函数
func stringPtr(s string) *string {
	return &s
}

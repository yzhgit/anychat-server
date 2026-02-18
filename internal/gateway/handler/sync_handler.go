package handler

import (
	"strconv"

	syncpb "github.com/anychat/server/api/proto/sync"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// SyncHandler sync HTTP处理器
type SyncHandler struct {
	clientManager *client.Manager
}

// NewSyncHandler 创建sync处理器
func NewSyncHandler(clientManager *client.Manager) *SyncHandler {
	return &SyncHandler{clientManager: clientManager}
}

// syncRequest 全量/增量同步请求体
type syncRequest struct {
	LastSyncTime     int64                  `json:"lastSyncTime"`
	ConversationSeqs []*conversationSeqItem `json:"conversationSeqs"`
}

type conversationSeqItem struct {
	ConversationId   string `json:"conversationId"`
	ConversationType string `json:"conversationType"`
	LastSeq          int64  `json:"lastSeq"`
}

// Sync 全量/增量数据同步
// @Summary      数据同步
// @Description  客户端登录或从后台恢复时调用，返回自 lastSyncTime 后的好友、群组、会话变更，以及各会话的离线消息。lastSyncTime=0 表示全量同步。
// @Tags         同步
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      syncRequest  true  "同步请求"
// @Success      200      {object}  response.Response{data=object}  "同步成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /sync [post]
func (h *SyncHandler) Sync(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req syncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = syncRequest{}
	}

	pbSeqs := make([]*syncpb.ConversationSeq, 0, len(req.ConversationSeqs))
	for _, s := range req.ConversationSeqs {
		pbSeqs = append(pbSeqs, &syncpb.ConversationSeq{
			ConversationId:   s.ConversationId,
			ConversationType: s.ConversationType,
			LastSeq:          s.LastSeq,
		})
	}

	resp, err := h.clientManager.Sync().Sync(c.Request.Context(), &syncpb.SyncRequest{
		UserId:           userID,
		LastSyncTime:     req.LastSyncTime,
		ConversationSeqs: pbSeqs,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// syncMessagesRequest 消息补齐请求体
type syncMessagesRequest struct {
	ConversationSeqs    []*conversationSeqItem `json:"conversationSeqs"`
	LimitPerConversation int32                 `json:"limitPerConversation"`
}

// SyncMessages 消息补齐
// @Summary      消息补齐
// @Description  按照每个会话的最新已知序列号，拉取离线消息
// @Tags         同步
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit  query  int  false  "每个会话最多返回的消息数（默认50）"
// @Param        request  body  syncMessagesRequest  true  "消息补齐请求"
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /sync/messages [post]
func (h *SyncHandler) SyncMessages(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req syncMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleGRPCError(c, err)
		return
	}

	limit := req.LimitPerConversation
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = int32(l)
		}
	}

	pbSeqs := make([]*syncpb.ConversationSeq, 0, len(req.ConversationSeqs))
	for _, s := range req.ConversationSeqs {
		pbSeqs = append(pbSeqs, &syncpb.ConversationSeq{
			ConversationId:   s.ConversationId,
			ConversationType: s.ConversationType,
			LastSeq:          s.LastSeq,
		})
	}

	resp, err := h.clientManager.Sync().SyncMessages(c.Request.Context(), &syncpb.SyncMessagesRequest{
		UserId:               userID,
		ConversationSeqs:     pbSeqs,
		LimitPerConversation: limit,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

package handler

import (
	"strconv"

	syncpb "github.com/anychat/server/api/proto/sync"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// SyncHandler sync HTTP handler
type SyncHandler struct {
	clientManager *client.Manager
}

// NewSyncHandler creates sync handler
func NewSyncHandler(clientManager *client.Manager) *SyncHandler {
	return &SyncHandler{clientManager: clientManager}
}

// syncRequest full/incremental sync request body
type syncRequest struct {
	LastSyncTime     int64                  `json:"lastSyncTime"`
	ConversationSeqs []*conversationSeqItem `json:"conversationSeqs"`
}

type conversationSeqItem struct {
	ConversationId   string `json:"conversationId"`
	ConversationType string `json:"conversationType"`
	LastSeq          int64  `json:"lastSeq"`
}

// Sync full/incremental data sync
// @Summary      data sync
// @Description  Called when client logs in or resumes from background. Returns friends, groups, conversation changes since lastSyncTime, and offline messages for each conversation. lastSyncTime=0 means full sync.
// @Tags         sync
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      syncRequest  true  "sync request"
// @Success      200      {object}  response.Response{data=object}  "sync success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      500      {object}  response.Response  "server error"
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

// syncMessagesRequest message sync request body
type syncMessagesRequest struct {
	ConversationSeqs     []*conversationSeqItem `json:"conversationSeqs"`
	LimitPerConversation int32                  `json:"limitPerConversation"`
}

// SyncMessages message sync
// @Summary      message sync
// @Description  Pull offline messages based on the latest known sequence number for each conversation
// @Tags         sync
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit  query  int  false  "max messages per conversation (default 50)"
// @Param        request  body  syncMessagesRequest  true  "message sync request"
// @Success      200  {object}  response.Response{data=object}  "success"
// @Failure      400  {object}  response.Response  "parameter error"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

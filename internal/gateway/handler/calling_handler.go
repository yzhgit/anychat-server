package handler

import (
	"net/http"
	"strconv"

	callingpb "github.com/anychat/server/api/proto/calling"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// CallingHandler audio/video call HTTP handler
type CallingHandler struct {
	clientManager *client.Manager
}

// NewCallingHandler creates Calling handler
func NewCallingHandler(clientManager *client.Manager) *CallingHandler {
	return &CallingHandler{clientManager: clientManager}
}

// ── One-on-one call ────────────────────────────────────────────

// initiateCallRequest initiate call request body
type initiateCallRequest struct {
	CalleeID string `json:"calleeId" binding:"required"`
	CallType string `json:"callType"` // audio/video (default: audio)
}

// InitiateCall initiate audio/video call
// @Summary      initiate call
// @Description  Initiate audio/video call to specified user, returns Calling Room name and JWT Token
// @Tags         Calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      initiateCallRequest  true  "call request"
// @Success      200      {object}  response.Response{data=object}  "call initiated successfully"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /calling/calls [post]
func (h *CallingHandler) InitiateCall(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req initiateCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CallType == "" {
		req.CallType = "audio"
	}

	callType := callingpb.CallType_CALL_TYPE_AUDIO
	if req.CallType == "video" {
		callType = callingpb.CallType_CALL_TYPE_VIDEO
	}

	resp, err := h.clientManager.Calling().InitiateCall(c.Request.Context(), &callingpb.InitiateCallRequest{
		CallerId: userID,
		CalleeId: req.CalleeID,
		CallType: callType,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// JoinCall answer call (callee accepts)
// @Summary      answer call
// @Description  Callee accepts call invitation, returns Calling Room name and JWT Token
// @Tags         Calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        callId  path  string  true  "call ID"
// @Success      200     {object}  response.Response{data=object}  "answer success"
// @Failure      400     {object}  response.Response  "parameter error"
// @Failure      401     {object}  response.Response  "unauthorized"
// @Failure      404     {object}  response.Response  "call not found"
// @Failure      500     {object}  response.Response  "server error"
// @Router       /calling/calls/{callId}/join [post]
func (h *CallingHandler) JoinCall(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	callID := c.Param("callId")

	resp, err := h.clientManager.Calling().JoinCall(c.Request.Context(), &callingpb.JoinCallRequest{
		CallId: callID,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// RejectCall reject call
// @Summary      reject call
// @Description  Callee rejects call invitation
// @Tags         Calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        callId  path  string  true  "call ID"
// @Success      200     {object}  response.Response  "reject success"
// @Failure      401     {object}  response.Response  "unauthorized"
// @Failure      404     {object}  response.Response  "call not found"
// @Failure      500     {object}  response.Response  "server error"
// @Router       /calling/calls/{callId}/reject [post]
func (h *CallingHandler) RejectCall(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	callID := c.Param("callId")

	_, err := h.clientManager.Calling().RejectCall(c.Request.Context(), &callingpb.RejectCallRequest{
		CallId: callID,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// EndCall end call
// @Summary      end call
// @Description  Caller or callee ends the call
// @Tags         Calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        callId  path  string  true  "call ID"
// @Success      200     {object}  response.Response  "end success"
// @Failure      401     {object}  response.Response  "unauthorized"
// @Failure      404     {object}  response.Response  "call not found"
// @Failure      500     {object}  response.Response  "server error"
// @Router       /calling/calls/{callId}/end [post]
func (h *CallingHandler) EndCall(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	callID := c.Param("callId")

	_, err := h.clientManager.Calling().EndCall(c.Request.Context(), &callingpb.EndCallRequest{
		CallId: callID,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetCallSession get call session details
// @Summary      get call details
// @Description  Get detailed info of specified call session
// @Tags         Calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        callId  path  string  true  "call ID"
// @Success      200     {object}  response.Response{data=object}  "success"
// @Failure      401     {object}  response.Response  "unauthorized"
// @Failure      404     {object}  response.Response  "call not found"
// @Failure      500     {object}  response.Response  "server error"
// @Router       /calling/calls/{callId} [get]
func (h *CallingHandler) GetCallSession(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	callID := c.Param("callId")

	resp, err := h.clientManager.Calling().GetCallSession(c.Request.Context(), &callingpb.GetCallSessionRequest{
		CallId: callID,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// ListCallLogs get call logs
// @Summary      call logs
// @Description  Get current user's call history
// @Tags         Calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page      query  int  false  "page number (default 1)"
// @Param        pageSize  query  int  false  "page size (default 20)"
// @Success      200       {object}  response.Response{data=object}  "success"
// @Failure      401       {object}  response.Response  "unauthorized"
// @Failure      500       {object}  response.Response  "server error"
// @Router       /calling/calls [get]
func (h *CallingHandler) ListCallLogs(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	req := &callingpb.ListCallLogsRequest{UserId: userID, Page: 1, PageSize: 20}
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			req.Page = int32(v)
		}
	}
	if ps := c.Query("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil {
			req.PageSize = int32(v)
		}
	}

	resp, err := h.clientManager.Calling().ListCallLogs(c.Request.Context(), req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// ── Meeting room ────────────────────────────────────────────────

// createMeetingRequest create meeting room request body
type createMeetingRequest struct {
	Title           string `json:"title" binding:"required"`
	Password        string `json:"password"`
	MaxParticipants int32  `json:"maxParticipants"`
}

// CreateMeeting create meeting room
// @Summary      create meeting room
// @Description  Create new audio/video meeting room, returns meeting info and Calling Token
// @Tags         Calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      createMeetingRequest  true  "meeting room info"
// @Success      200      {object}  response.Response{data=object}  "create success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /calling/meetings [post]
func (h *CallingHandler) CreateMeeting(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req createMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.clientManager.Calling().CreateMeeting(c.Request.Context(), &callingpb.CreateMeetingRequest{
		CreatorId:       userID,
		Title:           req.Title,
		Password:        req.Password,
		MaxParticipants: req.MaxParticipants,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// ListMeetings list active meeting rooms
// @Summary      meeting room list
// @Description  Get list of currently active meeting rooms
// @Tags         Calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page      query  int  false  "page number (default 1)"
// @Param        pageSize  query  int  false  "page size (default 20)"
// @Success      200       {object}  response.Response{data=object}  "success"
// @Failure      401       {object}  response.Response  "unauthorized"
// @Failure      500       {object}  response.Response  "server error"
// @Router       /calling/meetings [get]
func (h *CallingHandler) ListMeetings(c *gin.Context) {
	req := &callingpb.ListMeetingsRequest{Page: 1, PageSize: 20}
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			req.Page = int32(v)
		}
	}
	if ps := c.Query("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil {
			req.PageSize = int32(v)
		}
	}

	resp, err := h.clientManager.Calling().ListMeetings(c.Request.Context(), req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// GetMeeting get meeting room details
// @Summary      get meeting room
// @Description  Get detailed info of specified meeting room
// @Tags         Calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        roomId  path  string  true  "meeting room ID"
// @Success      200     {object}  response.Response{data=object}  "success"
// @Failure      401     {object}  response.Response  "unauthorized"
// @Failure      404     {object}  response.Response  "meeting room not found"
// @Failure      500     {object}  response.Response  "server error"
// @Router       /calling/meetings/{roomId} [get]
func (h *CallingHandler) GetMeeting(c *gin.Context) {
	roomID := c.Param("roomId")

	resp, err := h.clientManager.Calling().GetMeeting(c.Request.Context(), &callingpb.GetMeetingRequest{
		RoomId: roomID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// joinMeetingRequest join meeting room request body
type joinMeetingRequest struct {
	Password string `json:"password"`
}

// JoinMeeting join meeting room
// @Summary      join meeting room
// @Description  Join specified meeting room, returns Calling Token
// @Tags         Calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        roomId   path  string              true  "meeting room ID"
// @Param        request  body  joinMeetingRequest  false "meeting room password (if any)"
// @Success      200      {object}  response.Response{data=object}  "join success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      403      {object}  response.Response  "wrong password"
// @Failure      404      {object}  response.Response  "meeting room not found"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /calling/meetings/{roomId}/join [post]
func (h *CallingHandler) JoinMeeting(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	roomID := c.Param("roomId")

	var req joinMeetingRequest
	_ = c.ShouldBindJSON(&req)

	resp, err := h.clientManager.Calling().JoinMeeting(c.Request.Context(), &callingpb.JoinMeetingRequest{
		UserId:   userID,
		RoomId:   roomID,
		Password: req.Password,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// EndMeeting end meeting room
// @Summary      end meeting room
// @Description  Creator ends meeting room, all participants will be removed after room closes
// @Tags         Calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        roomId  path  string  true  "meeting room ID"
// @Success      200     {object}  response.Response  "success"
// @Failure      401     {object}  response.Response  "unauthorized"
// @Failure      403     {object}  response.Response  "no permission"
// @Failure      404     {object}  response.Response  "meeting room not found"
// @Failure      500     {object}  response.Response  "server error"
// @Router       /calling/meetings/{roomId}/end [post]
func (h *CallingHandler) EndMeeting(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	roomID := c.Param("roomId")

	_, err := h.clientManager.Calling().EndMeeting(c.Request.Context(), &callingpb.EndMeetingRequest{
		RoomId:    roomID,
		CreatorId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

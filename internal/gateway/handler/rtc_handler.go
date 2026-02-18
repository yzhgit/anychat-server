package handler

import (
	"net/http"
	"strconv"

	rtcpb "github.com/anychat/server/api/proto/rtc"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// RTCHandler RTC 音视频 HTTP 处理器
type RTCHandler struct {
	clientManager *client.Manager
}

// NewRTCHandler 创建 RTC 处理器
func NewRTCHandler(clientManager *client.Manager) *RTCHandler {
	return &RTCHandler{clientManager: clientManager}
}

// ── 一对一通话 ────────────────────────────────────────────

// initiateCallRequest 发起通话请求体
type initiateCallRequest struct {
	CalleeID string `json:"calleeId" binding:"required"`
	CallType string `json:"callType"` // audio/video（默认 audio）
}

// InitiateCall 发起音视频通话
// @Summary      发起通话
// @Description  向指定用户发起音视频通话，返回 RTC Room 名称和 JWT Token
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      initiateCallRequest  true  "通话请求"
// @Success      200      {object}  response.Response{data=object}  "通话发起成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /rtc/calls [post]
func (h *RTCHandler) InitiateCall(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req initiateCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CallType == "" {
		req.CallType = "audio"
	}

	callType := rtcpb.CallType_CALL_TYPE_AUDIO
	if req.CallType == "video" {
		callType = rtcpb.CallType_CALL_TYPE_VIDEO
	}

	resp, err := h.clientManager.RTC().InitiateCall(c.Request.Context(), &rtcpb.InitiateCallRequest{
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

// JoinCall 接听通话（被叫方接受）
// @Summary      接听通话
// @Description  被叫方接受通话邀请，返回 RTC Room 名称和 JWT Token
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        callId  path  string  true  "通话ID"
// @Success      200     {object}  response.Response{data=object}  "接听成功"
// @Failure      400     {object}  response.Response  "参数错误"
// @Failure      401     {object}  response.Response  "未授权"
// @Failure      404     {object}  response.Response  "通话不存在"
// @Failure      500     {object}  response.Response  "服务器错误"
// @Router       /rtc/calls/{callId}/join [post]
func (h *RTCHandler) JoinCall(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	callID := c.Param("callId")

	resp, err := h.clientManager.RTC().JoinCall(c.Request.Context(), &rtcpb.JoinCallRequest{
		CallId: callID,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// RejectCall 拒绝通话
// @Summary      拒绝通话
// @Description  被叫方拒绝通话邀请
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        callId  path  string  true  "通话ID"
// @Success      200     {object}  response.Response  "拒绝成功"
// @Failure      401     {object}  response.Response  "未授权"
// @Failure      404     {object}  response.Response  "通话不存在"
// @Failure      500     {object}  response.Response  "服务器错误"
// @Router       /rtc/calls/{callId}/reject [post]
func (h *RTCHandler) RejectCall(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	callID := c.Param("callId")

	_, err := h.clientManager.RTC().RejectCall(c.Request.Context(), &rtcpb.RejectCallRequest{
		CallId: callID,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// EndCall 挂断通话
// @Summary      挂断通话
// @Description  主叫方或被叫方挂断通话
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        callId  path  string  true  "通话ID"
// @Success      200     {object}  response.Response  "挂断成功"
// @Failure      401     {object}  response.Response  "未授权"
// @Failure      404     {object}  response.Response  "通话不存在"
// @Failure      500     {object}  response.Response  "服务器错误"
// @Router       /rtc/calls/{callId}/end [post]
func (h *RTCHandler) EndCall(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	callID := c.Param("callId")

	_, err := h.clientManager.RTC().EndCall(c.Request.Context(), &rtcpb.EndCallRequest{
		CallId: callID,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetCallSession 获取通话会话详情
// @Summary      获取通话详情
// @Description  获取指定通话会话的详细信息
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        callId  path  string  true  "通话ID"
// @Success      200     {object}  response.Response{data=object}  "成功"
// @Failure      401     {object}  response.Response  "未授权"
// @Failure      404     {object}  response.Response  "通话不存在"
// @Failure      500     {object}  response.Response  "服务器错误"
// @Router       /rtc/calls/{callId} [get]
func (h *RTCHandler) GetCallSession(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	callID := c.Param("callId")

	resp, err := h.clientManager.RTC().GetCallSession(c.Request.Context(), &rtcpb.GetCallSessionRequest{
		CallId: callID,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// ListCallLogs 获取通话记录
// @Summary      通话记录
// @Description  获取当前用户的通话历史记录
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page      query  int  false  "页码（默认1）"
// @Param        pageSize  query  int  false  "每页数量（默认20）"
// @Success      200       {object}  response.Response{data=object}  "成功"
// @Failure      401       {object}  response.Response  "未授权"
// @Failure      500       {object}  response.Response  "服务器错误"
// @Router       /rtc/calls [get]
func (h *RTCHandler) ListCallLogs(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	req := &rtcpb.ListCallLogsRequest{UserId: userID, Page: 1, PageSize: 20}
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

	resp, err := h.clientManager.RTC().ListCallLogs(c.Request.Context(), req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// ── 会议室 ────────────────────────────────────────────────

// createMeetingRequest 创建会议室请求体
type createMeetingRequest struct {
	Title           string `json:"title" binding:"required"`
	Password        string `json:"password"`
	MaxParticipants int32  `json:"maxParticipants"`
}

// CreateMeeting 创建会议室
// @Summary      创建会议室
// @Description  创建新的音视频会议室，返回会议信息和 RTC Token
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      createMeetingRequest  true  "会议室信息"
// @Success      200      {object}  response.Response{data=object}  "创建成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /rtc/meetings [post]
func (h *RTCHandler) CreateMeeting(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req createMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.clientManager.RTC().CreateMeeting(c.Request.Context(), &rtcpb.CreateMeetingRequest{
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

// ListMeetings 列举活跃会议室
// @Summary      会议室列表
// @Description  获取当前活跃的会议室列表
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page      query  int  false  "页码（默认1）"
// @Param        pageSize  query  int  false  "每页数量（默认20）"
// @Success      200       {object}  response.Response{data=object}  "成功"
// @Failure      401       {object}  response.Response  "未授权"
// @Failure      500       {object}  response.Response  "服务器错误"
// @Router       /rtc/meetings [get]
func (h *RTCHandler) ListMeetings(c *gin.Context) {
	req := &rtcpb.ListMeetingsRequest{Page: 1, PageSize: 20}
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

	resp, err := h.clientManager.RTC().ListMeetings(c.Request.Context(), req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// GetMeeting 获取会议室详情
// @Summary      获取会议室
// @Description  获取指定会议室的详细信息
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        roomId  path  string  true  "会议室ID"
// @Success      200     {object}  response.Response{data=object}  "成功"
// @Failure      401     {object}  response.Response  "未授权"
// @Failure      404     {object}  response.Response  "会议室不存在"
// @Failure      500     {object}  response.Response  "服务器错误"
// @Router       /rtc/meetings/{roomId} [get]
func (h *RTCHandler) GetMeeting(c *gin.Context) {
	roomID := c.Param("roomId")

	resp, err := h.clientManager.RTC().GetMeeting(c.Request.Context(), &rtcpb.GetMeetingRequest{
		RoomId: roomID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// joinMeetingRequest 加入会议室请求体
type joinMeetingRequest struct {
	Password string `json:"password"`
}

// JoinMeeting 加入会议室
// @Summary      加入会议室
// @Description  加入指定会议室，返回 RTC Token
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        roomId   path  string              true  "会议室ID"
// @Param        request  body  joinMeetingRequest  false "会议室密码（若有）"
// @Success      200      {object}  response.Response{data=object}  "加入成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      403      {object}  response.Response  "密码错误"
// @Failure      404      {object}  response.Response  "会议室不存在"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /rtc/meetings/{roomId}/join [post]
func (h *RTCHandler) JoinMeeting(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	roomID := c.Param("roomId")

	var req joinMeetingRequest
	_ = c.ShouldBindJSON(&req)

	resp, err := h.clientManager.RTC().JoinMeeting(c.Request.Context(), &rtcpb.JoinMeetingRequest{
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

// EndMeeting 结束会议室
// @Summary      结束会议室
// @Description  创建者结束会议室，会议室关闭后所有参与者将被移出
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        roomId  path  string  true  "会议室ID"
// @Success      200     {object}  response.Response  "成功"
// @Failure      401     {object}  response.Response  "未授权"
// @Failure      403     {object}  response.Response  "无权限"
// @Failure      404     {object}  response.Response  "会议室不存在"
// @Failure      500     {object}  response.Response  "服务器错误"
// @Router       /rtc/meetings/{roomId}/end [post]
func (h *RTCHandler) EndMeeting(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	roomID := c.Param("roomId")

	_, err := h.clientManager.RTC().EndMeeting(c.Request.Context(), &rtcpb.EndMeetingRequest{
		RoomId:    roomID,
		CreatorId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

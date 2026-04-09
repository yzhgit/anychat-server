package handler

import (
	"strconv"

	grouppb "github.com/anychat/server/api/proto/group"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	groupdto "github.com/anychat/server/internal/group/dto"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type GroupHandler struct {
	clientManager *client.Manager
}

func NewGroupHandler(clientManager *client.Manager) *GroupHandler {
	return &GroupHandler{clientManager: clientManager}
}

// CreateGroup create group
// @Summary      create group
// @Description  Create a new group, requires at least 2 members (including creator)
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      groupdto.CreateGroupRequest  true  "group info"
// @Success      200      {object}  response.Response{data=groupdto.GroupResponse}  "create success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /groups [post]
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req groupdto.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.clientManager.Group().CreateGroup(c.Request.Context(), &grouppb.CreateGroupRequest{
		UserId:    userID,
		Name:      req.Name,
		Avatar:    &req.Avatar,
		MemberIds: req.MemberIDs,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	// Convert proto response to DTO response for proper camelCase JSON
	avatar := ""
	if resp.Avatar != nil {
		avatar = *resp.Avatar
	}

	groupResp := &groupdto.GroupResponse{
		GroupID:     resp.GroupId,
		Name:        resp.Name,
		Avatar:      avatar,
		OwnerID:     resp.OwnerId,
		MemberCount: resp.MemberCount,
		MyRole:      "owner", // Creator is always the owner
		CreatedAt:   resp.CreatedAt.AsTime(),
	}

	response.Success(c, groupResp)
}

// GetGroupInfo get group info
// @Summary      get group info
// @Description  Get detailed info of specified group
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "group ID"
// @Success      200  {object}  response.Response{data=groupdto.GroupResponse}
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      404  {object}  response.Response  "group not found"
// @Router       /groups/{id} [get]
func (h *GroupHandler) GetGroupInfo(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	resp, err := h.clientManager.Group().GetGroupInfo(c.Request.Context(), &grouppb.GetGroupInfoRequest{
		GroupId: groupID,
		UserId:  &userID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	result := groupdto.GroupResponse{
		GroupID:      resp.GroupId,
		Name:         resp.Name,
		DisplayName:  resp.DisplayName,
		Avatar:       resp.Avatar,
		Announcement: resp.Announcement,
		Description:  resp.Description,
		OwnerID:      resp.OwnerId,
		MemberCount:  resp.MemberCount,
		MaxMembers:   resp.MaxMembers,
		JoinVerify:   resp.JoinVerify,
		IsMuted:      resp.IsMuted,
		CreatedAt:    resp.CreatedAt.AsTime(),
		UpdatedAt:    resp.UpdatedAt.AsTime(),
	}

	// Get user's role in the group
	if memberResp, _ := h.clientManager.Group().IsMember(c.Request.Context(), &grouppb.IsMemberRequest{
		GroupId: groupID,
		UserId:  userID,
	}); memberResp != nil && memberResp.IsMember {
		result.MyRole = memberResp.Role
	}

	response.Success(c, result)
}

// UpdateGroup update group info
// @Summary      update group info
// @Description  Update group name, avatar, announcement etc (requires admin permission)
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                        true  "group ID"
// @Param        request  body      groupdto.UpdateGroupRequest  true  "update info"
// @Success      200      {object}  response.Response  "update success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      403      {object}  response.Response  "no permission"
// @Router       /groups/{id} [put]
func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	var req groupdto.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err := h.clientManager.Group().UpdateGroup(c.Request.Context(), &grouppb.UpdateGroupRequest{
		UserId:       userID,
		GroupId:      groupID,
		Name:         req.Name,
		Avatar:       req.Avatar,
		Announcement: req.Announcement,
		Description:  req.Description,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// DissolveGroup dissolve group
// @Summary      dissolve group
// @Description  Dissolve group (owner only)
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "group ID"
// @Success      200  {object}  response.Response  "dissolve success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      403  {object}  response.Response  "no permission"
// @Router       /groups/{id} [delete]
func (h *GroupHandler) DissolveGroup(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	_, err := h.clientManager.Group().DissolveGroup(c.Request.Context(), &grouppb.DissolveGroupRequest{
		UserId:  userID,
		GroupId: groupID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// GetMyGroups get my groups
// @Summary      get my groups
// @Description  Get all groups the current user has joined
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        lastUpdateTime  query  int  false  "last update timestamp (incremental sync)"
// @Success      200  {object}  response.Response{data=groupdto.GroupListResponse}
// @Failure      401  {object}  response.Response  "unauthorized"
// @Router       /groups [get]
func (h *GroupHandler) GetMyGroups(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var lastUpdateTime *int64
	if lut := c.Query("lastUpdateTime"); lut != "" {
		if val, err := strconv.ParseInt(lut, 10, 64); err == nil {
			lastUpdateTime = &val
		}
	}

	resp, err := h.clientManager.Group().GetUserGroups(c.Request.Context(), &grouppb.GetUserGroupsRequest{
		UserId:         userID,
		LastUpdateTime: lastUpdateTime,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// GetGroupMembers get group members
// @Summary      get group members
// @Description  Get members of specified group, supports pagination
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id        path   string  true   "group ID"
// @Param        page      query  int     false  "page number"  default(1)
// @Param        pageSize  query  int     false  "page size"  default(20)
// @Success      200  {object}  response.Response{data=groupdto.GroupMemberListResponse}
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      404  {object}  response.Response  "group not found"
// @Router       /groups/{id}/members [get]
func (h *GroupHandler) GetGroupMembers(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	if ps := c.Query("pageSize"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 && val <= 100 {
			pageSize = val
		}
	}

	resp, err := h.clientManager.Group().GetGroupMembers(c.Request.Context(), &grouppb.GetGroupMembersRequest{
		UserId:   userID,
		GroupId:  groupID,
		Page:     int32Ptr(int32(page)),
		PageSize: int32Ptr(int32(pageSize)),
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// InviteMembers invite members
// @Summary      invite members
// @Description  Invite users to join group
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                         true  "group ID"
// @Param        request  body  groupdto.InviteMembersRequest  true  "invite info"
// @Success      200      {object}  response.Response  "invite success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Router       /groups/{id}/members [post]
func (h *GroupHandler) InviteMembers(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	var req groupdto.InviteMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err := h.clientManager.Group().InviteMembers(c.Request.Context(), &grouppb.InviteMembersRequest{
		UserId:     userID,
		GroupId:    groupID,
		InviteeIds: req.UserIDs,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// RemoveMember remove member
// @Summary      remove member
// @Description  Remove member from group (requires admin permission)
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path  string  true  "group ID"
// @Param        userId  path  string  true  "user ID to remove"
// @Success      200     {object}  response.Response  "remove success"
// @Failure      401     {object}  response.Response  "unauthorized"
// @Failure      403     {object}  response.Response  "no permission"
// @Router       /groups/{id}/members/{userId} [delete]
func (h *GroupHandler) RemoveMember(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")
	targetUserID := c.Param("userId")

	_, err := h.clientManager.Group().RemoveMember(c.Request.Context(), &grouppb.RemoveMemberRequest{
		UserId:       userID,
		GroupId:      groupID,
		TargetUserId: targetUserID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// QuitGroup quit group
// @Summary      quit group
// @Description  Quit specified group
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "group ID"
// @Success      200  {object}  response.Response  "quit success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Router       /groups/{id}/quit [post]
func (h *GroupHandler) QuitGroup(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	_, err := h.clientManager.Group().QuitGroup(c.Request.Context(), &grouppb.QuitGroupRequest{
		UserId:  userID,
		GroupId: groupID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateMemberRole update member role
// @Summary      update member role
// @Description  Set or remove group admin (owner only)
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                              true  "group ID"
// @Param        userId   path  string                              true  "user ID"
// @Param        request  body  groupdto.UpdateMemberRoleRequest  true  "role info"
// @Success      200      {object}  response.Response  "update success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      403      {object}  response.Response  "no permission"
// @Router       /groups/{id}/members/{userId}/role [put]
func (h *GroupHandler) UpdateMemberRole(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")
	targetUserID := c.Param("userId")

	var req groupdto.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err := h.clientManager.Group().UpdateMemberRole(c.Request.Context(), &grouppb.UpdateMemberRoleRequest{
		UserId:       userID,
		GroupId:      groupID,
		TargetUserId: targetUserID,
		Role:         req.Role,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateMemberNickname update member nickname
// @Summary      update member nickname
// @Description  Set your nickname in the group
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                                  true  "group ID"
// @Param        request  body  groupdto.UpdateMemberNicknameRequest  true  "nickname info"
// @Success      200      {object}  response.Response  "update success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Router       /groups/{id}/nickname [put]
func (h *GroupHandler) UpdateMemberNickname(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	var req groupdto.UpdateMemberNicknameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err := h.clientManager.Group().UpdateMemberNickname(c.Request.Context(), &grouppb.UpdateMemberNicknameRequest{
		UserId:   userID,
		GroupId:  groupID,
		Nickname: req.Nickname,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// TransferOwnership transfer ownership
// @Summary      transfer ownership
// @Description  Transfer group owner to another member (owner only)
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                                true  "group ID"
// @Param        request  body  groupdto.TransferOwnershipRequest  true  "transfer info"
// @Success      200      {object}  response.Response  "transfer success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      403      {object}  response.Response  "no permission"
// @Router       /groups/{id}/transfer [post]
func (h *GroupHandler) TransferOwnership(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	var req groupdto.TransferOwnershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err := h.clientManager.Group().TransferOwnership(c.Request.Context(), &grouppb.TransferOwnershipRequest{
		UserId:     userID,
		GroupId:    groupID,
		NewOwnerId: req.NewOwnerID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// JoinGroup join group
// @Summary      join group
// @Description  Request to join specified group
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                      true  "group ID"
// @Param        request  body  groupdto.JoinGroupRequest  true  "join request"
// @Success      200      {object}  response.Response{data=groupdto.JoinGroupResponse}  "request success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Router       /groups/{id}/join [post]
func (h *GroupHandler) JoinGroup(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	var req groupdto.JoinGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.clientManager.Group().JoinGroup(c.Request.Context(), &grouppb.JoinGroupRequest{
		UserId:  userID,
		GroupId: groupID,
		Message: &req.Message,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	result := groupdto.JoinGroupResponse{
		NeedVerify: resp.NeedVerify,
		RequestID:  resp.RequestId,
	}

	if resp.NeedVerify {
		result.Message = "Request submitted, awaiting review"
	} else {
		result.Message = "Successfully joined group"
	}

	response.Success(c, result)
}

// HandleJoinRequest handle join request
// @Summary      handle join request
// @Description  Accept or reject join request (requires admin permission)
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  string                                true  "group ID"
// @Param        requestId  path  int                                   true  "request ID"
// @Param        request    body  groupdto.HandleJoinRequestRequest  true  "handle info"
// @Success      200        {object}  response.Response  "handle success"
// @Failure      400        {object}  response.Response  "parameter error"
// @Failure      401        {object}  response.Response  "unauthorized"
// @Failure      403        {object}  response.Response  "no permission"
// @Router       /groups/{id}/requests/{requestId} [put]
func (h *GroupHandler) HandleJoinRequest(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	requestID, err := strconv.ParseInt(c.Param("requestId"), 10, 64)
	if err != nil {
		response.ParamError(c, "Invalid request ID")
		return
	}

	var req groupdto.HandleJoinRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err = h.clientManager.Group().HandleJoinRequest(c.Request.Context(), &grouppb.HandleJoinRequestRequest{
		UserId:    userID,
		RequestId: requestID,
		Accept:    req.Accept,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// MuteMember mute member
func (h *GroupHandler) MuteMember(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")
	targetUserID := c.Param("userId")

	var req groupdto.MuteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	muteType := grouppb.MuteType_MUTE_TYPE_PERMANENT
	if req.Type == "temporary" {
		muteType = grouppb.MuteType_MUTE_TYPE_TEMPORARY
	}

	_, err := h.clientManager.Group().MuteMember(c.Request.Context(), &grouppb.MuteMemberRequest{
		UserId:          userID,
		GroupId:         groupID,
		TargetUserId:    targetUserID,
		Type:            muteType,
		DurationMinutes: req.DurationMinutes,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// UnmuteMember unban member
func (h *GroupHandler) UnmuteMember(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")
	targetUserID := c.Param("userId")

	_, err := h.clientManager.Group().UnmuteMember(c.Request.Context(), &grouppb.UnmuteMemberRequest{
		UserId:       userID,
		GroupId:      groupID,
		TargetUserId: targetUserID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetJoinRequests get join requests
// @Summary      get join requests
// @Description  Get join requests for specified group (requires admin permission)
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path   string  true   "group ID"
// @Param        status  query  string  false  "request status (pending/accepted/rejected)"
// @Success      200     {object}  response.Response{data=groupdto.JoinRequestListResponse}
// @Failure      401     {object}  response.Response  "unauthorized"
// @Failure      403     {object}  response.Response  "no permission"
// @Router       /groups/{id}/requests [get]
func (h *GroupHandler) GetJoinRequests(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	var status *string
	if s := c.Query("status"); s != "" {
		status = &s
	}

	resp, err := h.clientManager.Group().GetJoinRequests(c.Request.Context(), &grouppb.GetJoinRequestsRequest{
		GroupId: groupID,
		Status:  status,
		UserId:  userID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// PinGroupMessage pin group message
func (h *GroupHandler) PinGroupMessage(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	var req groupdto.PinGroupMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err := h.clientManager.Group().PinGroupMessage(c.Request.Context(), &grouppb.PinGroupMessageRequest{
		UserId:    userID,
		GroupId:   groupID,
		MessageId: req.MessageID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// UnpinGroupMessage unpin group message
func (h *GroupHandler) UnpinGroupMessage(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")
	messageID := c.Param("messageId")

	_, err := h.clientManager.Group().UnpinGroupMessage(c.Request.Context(), &grouppb.UnpinGroupMessageRequest{
		UserId:    userID,
		GroupId:   groupID,
		MessageId: messageID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetPinnedMessages get group pinned messages
func (h *GroupHandler) GetPinnedMessages(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	resp, err := h.clientManager.Group().GetPinnedMessages(c.Request.Context(), &grouppb.GetPinnedMessagesRequest{
		UserId:  userID,
		GroupId: groupID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// SetGroupMute set group mute
func (h *GroupHandler) SetGroupMute(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	var req groupdto.SetGroupMuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err := h.clientManager.Group().SetGroupMute(c.Request.Context(), &grouppb.SetGroupMuteRequest{
		UserId:  userID,
		GroupId: groupID,
		Enabled: *req.Enabled,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// UpdateGroupSettings update group settings
func (h *GroupHandler) UpdateGroupSettings(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	var req groupdto.UpdateGroupSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err := h.clientManager.Group().UpdateGroupSettings(c.Request.Context(), &grouppb.UpdateGroupSettingsRequest{
		UserId:            userID,
		GroupId:           groupID,
		JoinVerify:        req.JoinVerify,
		AllowMemberInvite: req.AllowMemberInvite,
		AllowViewHistory:  req.AllowViewHistory,
		AllowAddFriend:    req.AllowAddFriend,
		AllowMemberModify: req.AllowMemberModify,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetGroupSettings get group settings
func (h *GroupHandler) GetGroupSettings(c *gin.Context) {
	groupID := c.Param("id")
	resp, err := h.clientManager.Group().GetGroupSettings(c.Request.Context(), &grouppb.GetGroupSettingsRequest{
		GroupId: groupID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, gin.H{
		"groupId":           resp.GroupId,
		"joinVerify":        resp.JoinVerify,
		"allowMemberInvite": resp.AllowMemberInvite,
		"allowViewHistory":  resp.AllowViewHistory,
		"allowAddFriend":    resp.AllowAddFriend,
		"allowMemberModify": resp.AllowMemberModify,
	})
}

// Helper function to create int32 pointer
func int32Ptr(v int32) *int32 {
	return &v
}

// UpdateMemberRemark set/clear group remark
// @Summary      set group remark
// @Description  Set a remark for the group that only you can see, pass empty string to clear
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                              true  "group ID"
// @Param        request  body  groupdto.UpdateMemberRemarkRequest  true  "remark info"
// @Success      200      {object}  response.Response  "set success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      403      {object}  response.Response  "not a group member"
// @Router       /groups/{id}/remark [put]
func (h *GroupHandler) UpdateMemberRemark(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	var req groupdto.UpdateMemberRemarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err := h.clientManager.Group().UpdateMemberRemark(c.Request.Context(), &grouppb.UpdateMemberRemarkRequest{
		UserId:  userID,
		GroupId: groupID,
		Remark:  req.Remark,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// GetGroupQRCode get group QR code
// @Summary      get group QR code
// @Description  Get current valid group QR code, auto create/renew if not exists or expiring soon (all members)
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "group ID"
// @Success      200  {object}  response.Response{data=groupdto.GroupQRCodeResponse}
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      403  {object}  response.Response  "not a group member"
// @Router       /groups/{id}/qrcode [get]
func (h *GroupHandler) GetGroupQRCode(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	resp, err := h.clientManager.Group().GetGroupQRCode(c.Request.Context(), &grouppb.GetGroupQRCodeRequest{
		UserId:  userID,
		GroupId: groupID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, groupdto.GroupQRCodeResponse{
		Token:    resp.Token,
		DeepLink: resp.DeepLink,
		ExpireAt: resp.ExpireAt,
	})
}

// RefreshGroupQRCode refresh group QR code
// @Summary      refresh group QR code
// @Description  Refresh group QR code, invalidate old one and generate new one (owner/admin only)
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "group ID"
// @Success      200  {object}  response.Response{data=groupdto.GroupQRCodeResponse}
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      403  {object}  response.Response  "no permission"
// @Router       /groups/{id}/qrcode/refresh [post]
func (h *GroupHandler) RefreshGroupQRCode(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	resp, err := h.clientManager.Group().RefreshGroupQRCode(c.Request.Context(), &grouppb.RefreshGroupQRCodeRequest{
		UserId:  userID,
		GroupId: groupID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, groupdto.GroupQRCodeResponse{
		Token:    resp.Token,
		DeepLink: resp.DeepLink,
		ExpireAt: resp.ExpireAt,
	})
}

// GetGroupPreviewByQRCode get group preview via QR code (no auth required)
// @Summary      group preview
// @Description  Get group name, avatar, member count and join verification via QR code token
// @Tags         group
// @Accept       json
// @Produce      json
// @Param        token  query  string  true  "QR code token"
// @Success      200  {object}  response.Response{data=groupdto.GroupQRCodePreviewResponse}
// @Failure      400  {object}  response.Response  "invalid parameter or QR code expired"
// @Router       /groups/preview [get]
func (h *GroupHandler) GetGroupPreviewByQRCode(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.ParamError(c, "token is required")
		return
	}

	resp, err := h.clientManager.Group().GetGroupPreviewByQRCode(c.Request.Context(), &grouppb.GetGroupPreviewByQRCodeRequest{
		Token: token,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, groupdto.GroupQRCodePreviewResponse{
		GroupID:     resp.GroupId,
		Name:        resp.Name,
		Avatar:      resp.Avatar,
		MemberCount: resp.MemberCount,
		NeedVerify:  resp.NeedVerify,
	})
}

// JoinGroupByQRCode join group via QR code
// @Summary      join group via QR code
// @Description  Join group via QR code token, join directly or submit request based on group settings
// @Tags         group
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  object{token=string}  true  "QR code token"
// @Success      200      {object}  response.Response{data=groupdto.JoinGroupByQRCodeResponse}
// @Failure      400      {object}  response.Response  "QR code invalid or expired"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Router       /groups/join-by-qrcode [post]
func (h *GroupHandler) JoinGroupByQRCode(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var body struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.clientManager.Group().JoinGroupByQRCode(c.Request.Context(), &grouppb.JoinGroupByQRCodeRequest{
		UserId: userID,
		Token:  body.Token,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	result := groupdto.JoinGroupByQRCodeResponse{
		Joined:     resp.Joined,
		GroupID:    resp.GroupId,
		NeedVerify: resp.NeedVerify,
		RequestID:  resp.RequestId,
	}
	response.Success(c, result)
}

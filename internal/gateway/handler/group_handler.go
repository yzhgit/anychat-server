package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	grouppb "github.com/anychat/server/api/proto/group"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	groupdto "github.com/anychat/server/internal/group/dto"
	"github.com/anychat/server/pkg/response"
)

type GroupHandler struct {
	clientManager *client.Manager
}

func NewGroupHandler(clientManager *client.Manager) *GroupHandler {
	return &GroupHandler{clientManager: clientManager}
}

// CreateGroup 创建群组
// @Summary      创建群组
// @Description  创建一个新的群组，至少需要2个成员（包括创建者）
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      groupdto.CreateGroupRequest  true  "群组信息"
// @Success      200      {object}  response.Response{data=groupdto.GroupResponse}  "创建成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /groups [post]
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req groupdto.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.clientManager.Group().CreateGroup(c.Request.Context(), &grouppb.CreateGroupRequest{
		UserId:     userID,
		Name:       req.Name,
		Avatar:     &req.Avatar,
		MemberIds:  req.MemberIDs,
		JoinVerify: &req.JoinVerify,
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

// GetGroupInfo 获取群组信息
// @Summary      获取群组信息
// @Description  获取指定群组的详细信息
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "群组ID"
// @Success      200  {object}  response.Response{data=groupdto.GroupResponse}
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      404  {object}  response.Response  "群组不存在"
// @Router       /groups/{id} [get]
func (h *GroupHandler) GetGroupInfo(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	groupID := c.Param("id")

	resp, err := h.clientManager.Group().GetGroupInfo(c.Request.Context(), &grouppb.GetGroupInfoRequest{
		GroupId: groupID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	// Get user's role in the group
	memberResp, _ := h.clientManager.Group().IsMember(c.Request.Context(), &grouppb.IsMemberRequest{
		GroupId: groupID,
		UserId:  userID,
	})

	result := groupdto.GroupResponse{
		GroupID:      resp.GroupId,
		Name:         resp.Name,
		Avatar:       resp.Avatar,
		Announcement: resp.Announcement,
		OwnerID:      resp.OwnerId,
		MemberCount:  resp.MemberCount,
		MaxMembers:   resp.MaxMembers,
		JoinVerify:   resp.JoinVerify,
		IsMuted:      resp.IsMuted,
		CreatedAt:    resp.CreatedAt.AsTime(),
		UpdatedAt:    resp.UpdatedAt.AsTime(),
	}

	if memberResp != nil && memberResp.IsMember {
		result.MyRole = memberResp.Role
	}

	response.Success(c, result)
}

// UpdateGroup 更新群信息
// @Summary      更新群信息
// @Description  更新群组的名称、头像、公告等信息（需要管理员权限）
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                        true  "群组ID"
// @Param        request  body      groupdto.UpdateGroupRequest  true  "更新信息"
// @Success      200      {object}  response.Response  "更新成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      403      {object}  response.Response  "无权限"
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
		JoinVerify:   req.JoinVerify,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// DissolveGroup 解散群组
// @Summary      解散群组
// @Description  解散群组（仅群主可操作）
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "群组ID"
// @Success      200  {object}  response.Response  "解散成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      403  {object}  response.Response  "无权限"
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

// GetMyGroups 获取我的群组列表
// @Summary      获取我的群组列表
// @Description  获取当前用户加入的所有群组列表
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        lastUpdateTime  query  int  false  "最后更新时间戳（增量同步）"
// @Success      200  {object}  response.Response{data=groupdto.GroupListResponse}
// @Failure      401  {object}  response.Response  "未授权"
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

// GetGroupMembers 获取群成员列表
// @Summary      获取群成员列表
// @Description  获取指定群组的成员列表，支持分页
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id        path   string  true   "群组ID"
// @Param        page      query  int     false  "页码"  default(1)
// @Param        pageSize  query  int     false  "每页数量"  default(20)
// @Success      200  {object}  response.Response{data=groupdto.GroupMemberListResponse}
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      404  {object}  response.Response  "群组不存在"
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

// InviteMembers 邀请成员
// @Summary      邀请成员
// @Description  邀请用户加入群组
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                         true  "群组ID"
// @Param        request  body  groupdto.InviteMembersRequest  true  "邀请信息"
// @Success      200      {object}  response.Response  "邀请成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
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

// RemoveMember 移除成员
// @Summary      移除成员
// @Description  从群组中移除成员（需要管理员权限）
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path  string  true  "群组ID"
// @Param        userId  path  string  true  "要移除的用户ID"
// @Success      200     {object}  response.Response  "移除成功"
// @Failure      401     {object}  response.Response  "未授权"
// @Failure      403     {object}  response.Response  "无权限"
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

// QuitGroup 退出群组
// @Summary      退出群组
// @Description  退出指定群组
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "群组ID"
// @Success      200  {object}  response.Response  "退出成功"
// @Failure      401  {object}  response.Response  "未授权"
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

// UpdateMemberRole 更新成员角色
// @Summary      更新成员角色
// @Description  设置或取消群管理员（仅群主可操作）
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                              true  "群组ID"
// @Param        userId   path  string                              true  "用户ID"
// @Param        request  body  groupdto.UpdateMemberRoleRequest  true  "角色信息"
// @Success      200      {object}  response.Response  "更新成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      403      {object}  response.Response  "无权限"
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

// UpdateMemberNickname 更新群昵称
// @Summary      更新群昵称
// @Description  设置自己在群内的昵称
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                                  true  "群组ID"
// @Param        request  body  groupdto.UpdateMemberNicknameRequest  true  "昵称信息"
// @Success      200      {object}  response.Response  "更新成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
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

// TransferOwnership 转让群主
// @Summary      转让群主
// @Description  将群主转让给其他成员（仅群主可操作）
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                                true  "群组ID"
// @Param        request  body  groupdto.TransferOwnershipRequest  true  "转让信息"
// @Success      200      {object}  response.Response  "转让成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      403      {object}  response.Response  "无权限"
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

// JoinGroup 加入群组
// @Summary      加入群组
// @Description  申请加入指定群组
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string                      true  "群组ID"
// @Param        request  body  groupdto.JoinGroupRequest  true  "申请信息"
// @Success      200      {object}  response.Response{data=groupdto.JoinGroupResponse}  "申请成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
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
		result.Message = "申请已提交，等待审核"
	} else {
		result.Message = "成功加入群组"
	}

	response.Success(c, result)
}

// HandleJoinRequest 处理入群申请
// @Summary      处理入群申请
// @Description  接受或拒绝入群申请（需要管理员权限）
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id         path  string                                true  "群组ID"
// @Param        requestId  path  int                                   true  "申请ID"
// @Param        request    body  groupdto.HandleJoinRequestRequest  true  "处理信息"
// @Success      200        {object}  response.Response  "处理成功"
// @Failure      400        {object}  response.Response  "参数错误"
// @Failure      401        {object}  response.Response  "未授权"
// @Failure      403        {object}  response.Response  "无权限"
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

// GetJoinRequests 获取入群申请列表
// @Summary      获取入群申请列表
// @Description  获取指定群组的入群申请列表（需要管理员权限）
// @Tags         群组
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path   string  true   "群组ID"
// @Param        status  query  string  false  "申请状态（pending/accepted/rejected）"
// @Success      200     {object}  response.Response{data=groupdto.JoinRequestListResponse}
// @Failure      401     {object}  response.Response  "未授权"
// @Failure      403     {object}  response.Response  "无权限"
// @Router       /groups/{id}/requests [get]
func (h *GroupHandler) GetJoinRequests(c *gin.Context) {
	groupID := c.Param("id")

	var status *string
	if s := c.Query("status"); s != "" {
		status = &s
	}

	resp, err := h.clientManager.Group().GetJoinRequests(c.Request.Context(), &grouppb.GetJoinRequestsRequest{
		GroupId: groupID,
		Status:  status,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// Helper function to create int32 pointer
func int32Ptr(v int32) *int32 {
	return &v
}

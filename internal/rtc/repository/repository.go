package repository

import (
	"github.com/anychat/server/internal/rtc/model"
	"gorm.io/gorm"
)

// CallRepository 通话会话仓库
type CallRepository interface {
	CreateCallSession(session *model.CallSession) error
	GetCallSession(callID string) (*model.CallSession, error)
	UpdateCallSession(session *model.CallSession) error
	ListCallLogs(userID string, page, pageSize int) ([]*model.CallSession, int64, error)
}

type callRepository struct {
	db *gorm.DB
}

func NewCallRepository(db *gorm.DB) CallRepository {
	return &callRepository{db: db}
}

func (r *callRepository) CreateCallSession(session *model.CallSession) error {
	return r.db.Create(session).Error
}

func (r *callRepository) GetCallSession(callID string) (*model.CallSession, error) {
	var session model.CallSession
	err := r.db.Where("call_id = ?", callID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *callRepository) UpdateCallSession(session *model.CallSession) error {
	return r.db.Save(session).Error
}

func (r *callRepository) ListCallLogs(userID string, page, pageSize int) ([]*model.CallSession, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var sessions []*model.CallSession
	var total int64

	query := r.db.Model(&model.CallSession{}).
		Where("caller_id = ? OR callee_id = ?", userID, userID).
		Order("created_at DESC")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&sessions).Error; err != nil {
		return nil, 0, err
	}
	return sessions, total, nil
}

// MeetingRepository 会议室仓库
type MeetingRepository interface {
	CreateMeeting(meeting *model.MeetingRoom) error
	GetMeetingByRoomID(roomID string) (*model.MeetingRoom, error)
	UpdateMeeting(meeting *model.MeetingRoom) error
	ListActiveMeetings(page, pageSize int) ([]*model.MeetingRoom, int64, error)
}

type meetingRepository struct {
	db *gorm.DB
}

func NewMeetingRepository(db *gorm.DB) MeetingRepository {
	return &meetingRepository{db: db}
}

func (r *meetingRepository) CreateMeeting(meeting *model.MeetingRoom) error {
	return r.db.Create(meeting).Error
}

func (r *meetingRepository) GetMeetingByRoomID(roomID string) (*model.MeetingRoom, error) {
	var meeting model.MeetingRoom
	err := r.db.Where("room_id = ?", roomID).First(&meeting).Error
	if err != nil {
		return nil, err
	}
	return &meeting, nil
}

func (r *meetingRepository) UpdateMeeting(meeting *model.MeetingRoom) error {
	return r.db.Save(meeting).Error
}

func (r *meetingRepository) ListActiveMeetings(page, pageSize int) ([]*model.MeetingRoom, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var meetings []*model.MeetingRoom
	var total int64

	query := r.db.Model(&model.MeetingRoom{}).
		Where("status = ?", "active").
		Order("created_at DESC")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&meetings).Error; err != nil {
		return nil, 0, err
	}
	return meetings, total, nil
}

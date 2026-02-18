package model

import "time"

// CallSession 通话会话
type CallSession struct {
	ID          int64      `gorm:"primaryKey;autoIncrement"`
	CallID      string     `gorm:"column:call_id;uniqueIndex;not null"`
	CallerID    string     `gorm:"column:caller_id;not null;index"`
	CalleeID    string     `gorm:"column:callee_id;not null;index"`
	CallType    string     `gorm:"column:call_type;not null;default:audio"` // audio/video
	Status      string     `gorm:"column:status;not null;default:ringing"`  // ringing/connected/ended/rejected/missed/cancelled
	RoomName    string     `gorm:"column:room_name;not null"`
	StartedAt   time.Time  `gorm:"column:started_at;autoCreateTime"`
	ConnectedAt *time.Time `gorm:"column:connected_at"`
	EndedAt     *time.Time `gorm:"column:ended_at"`
	Duration    int        `gorm:"column:duration;not null;default:0"` // 秒
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (CallSession) TableName() string { return "call_sessions" }

// MeetingRoom 会议室
type MeetingRoom struct {
	ID              int64      `gorm:"primaryKey;autoIncrement"`
	RoomID          string     `gorm:"column:room_id;uniqueIndex;not null"`
	CreatorID       string     `gorm:"column:creator_id;not null;index"`
	Title           string     `gorm:"column:title;not null"`
	RoomName        string     `gorm:"column:room_name;uniqueIndex;not null"`
	PasswordHash    string     `gorm:"column:password_hash"`
	MaxParticipants int        `gorm:"column:max_participants;not null;default:0"`
	Status          string     `gorm:"column:status;not null;default:active"` // active/ended
	StartedAt       time.Time  `gorm:"column:started_at;autoCreateTime"`
	EndedAt         *time.Time `gorm:"column:ended_at"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (MeetingRoom) TableName() string { return "meeting_rooms" }

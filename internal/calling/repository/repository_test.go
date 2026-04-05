package repository

import (
	"testing"
	"time"

	"github.com/anychat/server/internal/calling/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCallingTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.CallSession{}, &model.MeetingRoom{})
	require.NoError(t, err)

	return db
}

func TestCallRepository_ListCallLogs_Empty(t *testing.T) {
	db := setupCallingTestDB(t)
	repo := NewCallRepository(db)

	sessions, total, err := repo.ListCallLogs("user-1", 1, 20)
	require.NoError(t, err)
	assert.Empty(t, sessions)
	assert.Equal(t, int64(0), total)
}

func TestCallRepository_ListCallLogs_SortedAndPaginated(t *testing.T) {
	db := setupCallingTestDB(t)
	repo := NewCallRepository(db)

	now := time.Now()
	fixtures := []*model.CallSession{
		{
			CallID:    "call-1",
			CallerID:  "user-1",
			CalleeID:  "user-2",
			CallType:  "audio",
			Status:    "ended",
			RoomName:  "call_room_1",
			CreatedAt: now.Add(-2 * time.Hour),
		},
		{
			CallID:    "call-2",
			CallerID:  "user-3",
			CalleeID:  "user-1",
			CallType:  "video",
			Status:    "ended",
			RoomName:  "call_room_2",
			CreatedAt: now.Add(-1 * time.Hour),
		},
		{
			CallID:    "call-3",
			CallerID:  "user-9",
			CalleeID:  "user-8",
			CallType:  "audio",
			Status:    "ended",
			RoomName:  "call_room_3",
			CreatedAt: now,
		},
	}
	for _, session := range fixtures {
		require.NoError(t, db.Create(session).Error)
	}

	sessions, total, err := repo.ListCallLogs("user-1", 1, 10)
	require.NoError(t, err)
	require.Len(t, sessions, 2)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, "call-2", sessions[0].CallID)
	assert.Equal(t, "call-1", sessions[1].CallID)
}

func TestMeetingRepository_ListActiveMeetings_Empty(t *testing.T) {
	db := setupCallingTestDB(t)
	repo := NewMeetingRepository(db)

	meetings, total, err := repo.ListActiveMeetings(1, 20)
	require.NoError(t, err)
	assert.Empty(t, meetings)
	assert.Equal(t, int64(0), total)
}

func TestMeetingRepository_ListActiveMeetings_FiltersAndSorts(t *testing.T) {
	db := setupCallingTestDB(t)
	repo := NewMeetingRepository(db)

	now := time.Now()
	fixtures := []*model.MeetingRoom{
		{
			RoomID:    "room-1",
			CreatorID: "user-1",
			Title:     "Older active meeting",
			RoomName:  "meeting_room_1",
			Status:    "active",
			CreatedAt: now.Add(-2 * time.Hour),
		},
		{
			RoomID:    "room-2",
			CreatorID: "user-2",
			Title:     "Newest active meeting",
			RoomName:  "meeting_room_2",
			Status:    "active",
			CreatedAt: now.Add(-30 * time.Minute),
		},
		{
			RoomID:    "room-3",
			CreatorID: "user-3",
			Title:     "Ended meeting",
			RoomName:  "meeting_room_3",
			Status:    "ended",
			CreatedAt: now,
		},
	}
	for _, meeting := range fixtures {
		require.NoError(t, db.Create(meeting).Error)
	}

	meetings, total, err := repo.ListActiveMeetings(1, 10)
	require.NoError(t, err)
	require.Len(t, meetings, 2)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, "room-2", meetings[0].RoomID)
	assert.Equal(t, "room-1", meetings[1].RoomID)
}

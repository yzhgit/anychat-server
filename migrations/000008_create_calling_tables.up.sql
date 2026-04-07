-- 通话会话表
CREATE TABLE IF NOT EXISTS call_sessions (
    id           BIGSERIAL    PRIMARY KEY,
    call_id      VARCHAR(36)  NOT NULL UNIQUE,
    caller_id    VARCHAR(36)  NOT NULL,
    callee_id    VARCHAR(36)  NOT NULL,
    call_type    VARCHAR(10)  NOT NULL DEFAULT 'audio',  -- audio/video
    status       VARCHAR(20)  NOT NULL DEFAULT 'ringing', -- ringing/connected/ended/rejected/missed/cancelled
    room_name    VARCHAR(100) NOT NULL,
    started_at   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    connected_at TIMESTAMP,
    ended_at     TIMESTAMP,
    duration     INT          NOT NULL DEFAULT 0,  -- 通话时长（秒）
    created_at   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_call_sessions_call_id   ON call_sessions (call_id);
CREATE INDEX idx_call_sessions_caller_id ON call_sessions (caller_id);
CREATE INDEX idx_call_sessions_callee_id ON call_sessions (callee_id);
CREATE INDEX idx_call_sessions_created_at ON call_sessions (created_at DESC);

-- 会议室表
CREATE TABLE IF NOT EXISTS meeting_rooms (
    id               BIGSERIAL    PRIMARY KEY,
    room_id          VARCHAR(36)  NOT NULL UNIQUE,
    creator_id       VARCHAR(36)  NOT NULL,
    title            VARCHAR(200) NOT NULL,
    room_name        VARCHAR(100) NOT NULL UNIQUE,  -- LiveKit Room 名称
    password_hash    VARCHAR(200),                  -- 可选密码哈希
    max_participants INT          NOT NULL DEFAULT 0,
    status           VARCHAR(20)  NOT NULL DEFAULT 'active',  -- active/ended
    started_at       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at         TIMESTAMP,
    created_at       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_meeting_rooms_room_id    ON meeting_rooms (room_id);
CREATE INDEX idx_meeting_rooms_creator_id ON meeting_rooms (creator_id);
CREATE INDEX idx_meeting_rooms_status     ON meeting_rooms (status);
CREATE INDEX idx_meeting_rooms_created_at ON meeting_rooms (created_at DESC);

-- 删除用户表
-- 先删依赖 users 的子表，再删 users，避免外键依赖报错
DROP TABLE IF EXISTS user_profiles;
DROP TABLE IF EXISTS user_settings;
DROP TABLE IF EXISTS user_qrcodes;
DROP TABLE IF EXISTS user_push_tokens;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS user_devices;
DROP TABLE IF EXISTS users;

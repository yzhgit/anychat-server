package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/anychat/server/internal/auth/dto"
	"github.com/anychat/server/internal/auth/model"
	"github.com/anychat/server/internal/auth/repository"
	pkgerrors "github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/logger"
	pkgredis "github.com/anychat/server/pkg/redis"
	"github.com/anychat/server/pkg/validator"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type VerificationService interface {
	SendCode(ctx context.Context, req *dto.SendCodeRequest, ipAddress string) (*dto.SendCodeResponse, error)
	VerifyCode(ctx context.Context, req *dto.VerifyCodeRequest) (*dto.VerifyCodeResponse, error)
	CheckCodeStatus(ctx context.Context, codeID string) (*dto.CheckCodeStatusResponse, error)
}

type verifyServiceImpl struct {
	codeRepo     repository.VerificationCodeRepository
	templateRepo repository.VerificationTemplateRepository
	cache        *pkgredis.Client
	smsSender    SMSSender
	emailSender  EmailSender
	config       Config
}

type Config struct {
	AppMode         string
	HashSecret      string
	CodeLength      int
	ExpireSeconds   int
	MaxAttempts     int
	TargetPerMinute int
	TargetPerDay    int
	IPPerHour       int
	DevicePerDay    int
	DebugFixedCode  string
	AllowDevBypass  bool
}

type SMSSender interface {
	Send(to, templateID, code string) error
}

type EmailSender interface {
	Send(to, subject, content string) error
}

func NewVerificationService(
	codeRepo repository.VerificationCodeRepository,
	templateRepo repository.VerificationTemplateRepository,
	cache *pkgredis.Client,
	smsSender SMSSender,
	emailSender EmailSender,
	config Config,
) VerificationService {
	if config.CodeLength == 0 {
		config.CodeLength = 6
	}
	if config.ExpireSeconds == 0 {
		config.ExpireSeconds = 300
	}
	if config.MaxAttempts == 0 {
		config.MaxAttempts = 5
	}
	if config.TargetPerMinute == 0 {
		config.TargetPerMinute = 1
	}
	if config.TargetPerDay == 0 {
		config.TargetPerDay = 10
	}
	if config.IPPerHour == 0 {
		config.IPPerHour = 200
	}
	if config.DevicePerDay == 0 {
		config.DevicePerDay = 100
	}

	return &verifyServiceImpl{
		codeRepo:     codeRepo,
		templateRepo: templateRepo,
		cache:        cache,
		smsSender:    smsSender,
		emailSender:  emailSender,
		config:       config,
	}
}

func (s *verifyServiceImpl) SendCode(ctx context.Context, req *dto.SendCodeRequest, ipAddress string) (*dto.SendCodeResponse, error) {
	target, err := s.validateAndNormalizeTarget(req.Target, req.TargetType)
	if err != nil {
		return nil, err
	}
	if err := s.validatePurpose(req.Purpose); err != nil {
		return nil, err
	}
	if s.cache == nil {
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "verification cache is not configured")
	}

	rollbackKeys, err := s.applyRateLimits(ctx, target, req.TargetType, req.Purpose, req.DeviceID, ipAddress)
	if err != nil {
		return nil, err
	}

	if err := s.cancelPreviousCode(ctx, target, req.TargetType, req.Purpose); err != nil {
		s.rollbackRateLimits(ctx, rollbackKeys)
		return nil, err
	}

	code := s.generateCode()
	codeID := fmt.Sprintf("vc_%d_%s", time.Now().UnixNano(), randString(8))
	expiresAt := time.Now().Add(time.Duration(s.config.ExpireSeconds) * time.Second)
	codeHash := s.hashCode(req.Purpose, target, code)

	cacheKey := s.codeCacheKey(req.Purpose, target)
	if err := s.cache.HSet(ctx, cacheKey,
		"code_id", codeID,
		"code_hash", codeHash,
		"target_type", req.TargetType,
		"expires_at", expiresAt.UTC().Format(time.RFC3339),
		"attempts", "0",
		"max_attempts", fmt.Sprintf("%d", s.config.MaxAttempts),
		"device_id", req.DeviceID,
	); err != nil {
		s.rollbackRateLimits(ctx, rollbackKeys)
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "")
	}
	if err := s.cache.Expire(ctx, cacheKey, time.Duration(s.config.ExpireSeconds)*time.Second); err != nil {
		_ = s.cache.Del(ctx, cacheKey)
		s.rollbackRateLimits(ctx, rollbackKeys)
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "")
	}

	record := &model.VerificationCode{
		CodeID:       codeID,
		Target:       target,
		TargetType:   req.TargetType,
		CodeHash:     codeHash,
		Purpose:      req.Purpose,
		ExpiresAt:    expiresAt,
		Status:       model.CodeStatusPending,
		SendIP:       ipAddress,
		SendDeviceID: req.DeviceID,
	}
	if err := s.codeRepo.Create(ctx, record); err != nil {
		_ = s.cache.Del(ctx, cacheKey)
		s.rollbackRateLimits(ctx, rollbackKeys)
		logger.Error("failed to create verification record", zap.Error(err))
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "")
	}

	if err := s.dispatchCode(ctx, target, req.TargetType, req.Purpose, code); err != nil {
		_ = s.cache.Del(ctx, cacheKey)
		s.rollbackRateLimits(ctx, rollbackKeys)
		_ = s.codeRepo.UpdateStatus(ctx, codeID, model.CodeStatusCancelled)
		return nil, err
	}

	return &dto.SendCodeResponse{
		CodeID:    codeID,
		ExpiresIn: int64(s.config.ExpireSeconds),
		Sent:      true,
		Message:   "verification code sent",
	}, nil
}

func (s *verifyServiceImpl) VerifyCode(ctx context.Context, req *dto.VerifyCodeRequest) (*dto.VerifyCodeResponse, error) {
	target, err := s.validateAndNormalizeTarget(req.Target, req.TargetType)
	if err != nil {
		return nil, err
	}
	if err := s.validatePurpose(req.Purpose); err != nil {
		return nil, err
	}
	if s.cache == nil {
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "verification cache is not configured")
	}

	cacheKey := s.codeCacheKey(req.Purpose, target)
	fields, err := s.cache.HGetAll(ctx, cacheKey)
	if err != nil {
		logger.Error("failed to load verification code from cache", zap.Error(err))
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "")
	}

	if len(fields) == 0 {
		if s.shouldAllowDevBypass(req.Code) {
			return &dto.VerifyCodeResponse{
				Valid:   true,
				CodeID:  "dev-bypass",
				Message: "verification successful",
			}, nil
		}
		return nil, s.resolveMissingCodeError(ctx, target, req.TargetType, req.Purpose)
	}

	codeID := fields["code_id"]
	if fields["target_type"] != req.TargetType {
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeVerifyCodeNotFound, "")
	}

	expiresAt, err := time.Parse(time.RFC3339, fields["expires_at"])
	if err != nil {
		logger.Error("invalid verification expiry in cache", zap.String("codeID", codeID), zap.Error(err))
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "")
	}
	if time.Now().After(expiresAt) {
		_ = s.cache.Del(ctx, cacheKey)
		_ = s.codeRepo.UpdateStatus(ctx, codeID, model.CodeStatusExpired)
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeVerifyCodeExpired, "")
	}

	if s.hashCode(req.Purpose, target, req.Code) != fields["code_hash"] {
		attempts, incrErr := s.cache.GetClient().HIncrBy(ctx, cacheKey, "attempts", 1).Result()
		if incrErr != nil {
			logger.Error("failed to increment verification attempts", zap.String("codeID", codeID), zap.Error(incrErr))
			return nil, pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "")
		}
		_ = s.codeRepo.IncrementAttempts(ctx, codeID)

		maxAttempts := int64(s.config.MaxAttempts)
		if fields["max_attempts"] != "" {
			if parsed, parseErr := parseInt64(fields["max_attempts"]); parseErr == nil {
				maxAttempts = parsed
			}
		}
		if attempts >= maxAttempts {
			_ = s.cache.Del(ctx, cacheKey)
			_ = s.codeRepo.UpdateStatus(ctx, codeID, model.CodeStatusLocked)
			return nil, pkgerrors.NewBusiness(pkgerrors.CodeVerifyAttemptsExceeded, "")
		}

		return nil, pkgerrors.NewBusiness(pkgerrors.CodeVerifyCodeError, "")
	}

	now := time.Now()
	if err := s.codeRepo.UpdateVerifiedAt(ctx, codeID, now); err != nil {
		logger.Error("failed to mark verification code as used", zap.String("codeID", codeID), zap.Error(err))
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "")
	}
	_ = s.cache.Del(ctx, cacheKey)

	return &dto.VerifyCodeResponse{
		Valid:   true,
		CodeID:  codeID,
		Message: "verification successful",
	}, nil
}

func (s *verifyServiceImpl) CheckCodeStatus(ctx context.Context, codeID string) (*dto.CheckCodeStatusResponse, error) {
	code, err := s.codeRepo.GetByCodeID(ctx, codeID)
	if err != nil {
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeVerifyCodeNotFound, "")
	}

	return &dto.CheckCodeStatusResponse{
		Status:    code.Status,
		ExpiresAt: code.ExpiresAt,
	}, nil
}

func (s *verifyServiceImpl) dispatchCode(ctx context.Context, target, targetType, purpose, code string) error {
	if !s.isReleaseMode() {
		logger.Info(
			"verification code generated for local environment",
			zap.String("target", maskTarget(target, targetType)),
			zap.String("targetType", targetType),
			zap.String("purpose", purpose),
			zap.String("code", code),
		)
	}

	templateID := ""
	emailSubject := "AnyChat Verification Code"
	emailContent := fmt.Sprintf("Your verification code is: %s, valid for 5 minutes.", code)
	if s.templateRepo != nil {
		template, err := s.templateRepo.GetByPurpose(ctx, purpose)
		if err == nil {
			templateID = template.SMSTemplateID
			if template.EmailSubject != "" {
				emailSubject = template.EmailSubject
			}
			if template.EmailContent != "" {
				emailContent = strings.ReplaceAll(template.EmailContent, "{code}", code)
			}
		}
	}

	switch targetType {
	case model.TargetTypeSMS:
		if s.smsSender == nil {
			if !s.isReleaseMode() {
				return nil
			}
			return pkgerrors.NewBusiness(pkgerrors.CodeSMSServiceError, "")
		}
		if err := s.smsSender.Send(target, templateID, code); err != nil {
			logger.Error("failed to send sms verification code", zap.Error(err))
			return pkgerrors.NewBusiness(pkgerrors.CodeSMSServiceError, "")
		}
	case model.TargetTypeEmail:
		if s.emailSender == nil {
			if !s.isReleaseMode() {
				return nil
			}
			return pkgerrors.NewBusiness(pkgerrors.CodeEmailServiceError, "")
		}
		if err := s.emailSender.Send(target, emailSubject, emailContent); err != nil {
			logger.Error("failed to send email verification code", zap.Error(err))
			return pkgerrors.NewBusiness(pkgerrors.CodeEmailServiceError, "")
		}
	default:
		return pkgerrors.NewBusiness(pkgerrors.CodeTargetFormatInvalid, "")
	}

	return nil
}

func (s *verifyServiceImpl) cancelPreviousCode(ctx context.Context, target, targetType, purpose string) error {
	code, err := s.codeRepo.GetLatestByTarget(ctx, target, targetType, purpose)
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		logger.Error("failed to query previous verification code", zap.Error(err))
		return pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "")
	}
	if code.Status == model.CodeStatusPending {
		if err := s.codeRepo.UpdateStatus(ctx, code.CodeID, model.CodeStatusCancelled); err != nil {
			logger.Error("failed to cancel previous verification code", zap.String("codeID", code.CodeID), zap.Error(err))
			return pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "")
		}
	}
	return nil
}

func (s *verifyServiceImpl) applyRateLimits(ctx context.Context, target, targetType, purpose, deviceID, ipAddress string) ([]string, error) {
	targetHash := s.targetHash(target)
	keys := make([]string, 0, 4)

	targetMinuteKey := fmt.Sprintf("auth:vc:rl:target:%s:%s:1m", purpose, targetHash)
	if err := s.incrementAndCheck(ctx, targetMinuteKey, time.Minute, s.config.TargetPerMinute, pkgerrors.CodeSendRateLimited); err != nil {
		return nil, err
	}
	keys = append(keys, targetMinuteKey)

	targetDayKey := fmt.Sprintf("auth:vc:rl:target:%s:%s:24h", purpose, targetHash)
	if err := s.incrementAndCheck(ctx, targetDayKey, 24*time.Hour, s.config.TargetPerDay, pkgerrors.CodeSendLimitReached); err != nil {
		s.rollbackRateLimits(ctx, keys)
		return nil, err
	}
	keys = append(keys, targetDayKey)

	if ipAddress != "" {
		ipKey := fmt.Sprintf("auth:vc:rl:ip:%s:1h", ipAddress)
		if err := s.incrementAndCheck(ctx, ipKey, time.Hour, s.config.IPPerHour, pkgerrors.CodeSendRateLimited); err != nil {
			s.rollbackRateLimits(ctx, keys)
			return nil, err
		}
		keys = append(keys, ipKey)
	}

	if deviceID != "" {
		deviceKey := fmt.Sprintf("auth:vc:rl:device:%s:24h", deviceID)
		if err := s.incrementAndCheck(ctx, deviceKey, 24*time.Hour, s.config.DevicePerDay, pkgerrors.CodeSendLimitReached); err != nil {
			s.rollbackRateLimits(ctx, keys)
			return nil, err
		}
		keys = append(keys, deviceKey)
	}

	return keys, nil
}

func (s *verifyServiceImpl) rollbackRateLimits(ctx context.Context, keys []string) {
	for _, key := range keys {
		if _, err := s.cache.Decr(ctx, key); err != nil {
			logger.Warn("failed to rollback verification rate limit counter", zap.String("key", key), zap.Error(err))
		}
	}
}

func (s *verifyServiceImpl) incrementAndCheck(ctx context.Context, key string, ttl time.Duration, limit int, errorCode int) error {
	current, err := s.cache.Incr(ctx, key)
	if err != nil {
		logger.Error("failed to increment verification rate limit counter", zap.String("key", key), zap.Error(err))
		return pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "")
	}
	if current == 1 {
		if err := s.cache.Expire(ctx, key, ttl); err != nil {
			logger.Error("failed to set verification rate limit ttl", zap.String("key", key), zap.Error(err))
			return pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "")
		}
	}
	if limit > 0 && current > int64(limit) {
		_, _ = s.cache.Decr(ctx, key)
		return pkgerrors.NewBusiness(errorCode, "")
	}
	return nil
}

func (s *verifyServiceImpl) resolveMissingCodeError(ctx context.Context, target, targetType, purpose string) error {
	code, err := s.codeRepo.GetLatestByTarget(ctx, target, targetType, purpose)
	if err == gorm.ErrRecordNotFound {
		return pkgerrors.NewBusiness(pkgerrors.CodeVerifyCodeNotFound, "")
	}
	if err != nil {
		logger.Error("failed to load verification audit record", zap.Error(err))
		return pkgerrors.NewBusiness(pkgerrors.CodeInternalError, "")
	}

	switch code.Status {
	case model.CodeStatusVerified:
		return pkgerrors.NewBusiness(pkgerrors.CodeVerifyCodeAlreadyUsed, "")
	case model.CodeStatusExpired:
		return pkgerrors.NewBusiness(pkgerrors.CodeVerifyCodeExpired, "")
	case model.CodeStatusLocked:
		return pkgerrors.NewBusiness(pkgerrors.CodeVerifyAttemptsExceeded, "")
	default:
		return pkgerrors.NewBusiness(pkgerrors.CodeVerifyCodeNotFound, "")
	}
}

func (s *verifyServiceImpl) validateAndNormalizeTarget(target, targetType string) (string, error) {
	normalized := strings.TrimSpace(target)
	switch targetType {
	case model.TargetTypeSMS:
		if !validator.ValidatePhone(normalized) {
			return "", pkgerrors.NewBusiness(pkgerrors.CodeTargetFormatInvalid, "invalid phone number format")
		}
	case model.TargetTypeEmail:
		normalized = strings.ToLower(normalized)
		if !validator.ValidateEmail(normalized) {
			return "", pkgerrors.NewBusiness(pkgerrors.CodeTargetFormatInvalid, "invalid email format")
		}
	default:
		return "", pkgerrors.NewBusiness(pkgerrors.CodeTargetFormatInvalid, "invalid target type")
	}
	return normalized, nil
}

func (s *verifyServiceImpl) validatePurpose(purpose string) error {
	switch purpose {
	case model.PurposeRegister,
		model.PurposeLogin,
		model.PurposeResetPassword,
		model.PurposeBindPhone,
		model.PurposeChangePhone,
		model.PurposeBindEmail,
		model.PurposeChangeEmail:
		return nil
	default:
		return pkgerrors.NewBusiness(pkgerrors.CodeParamError, "verification purpose not supported")
	}
}

func (s *verifyServiceImpl) codeCacheKey(purpose, target string) string {
	return fmt.Sprintf("auth:vc:%s:%s", purpose, s.targetHash(target))
}

func (s *verifyServiceImpl) targetHash(target string) string {
	sum := sha256.Sum256([]byte(target))
	return hex.EncodeToString(sum[:])
}

func (s *verifyServiceImpl) hashCode(purpose, target, code string) string {
	secret := s.config.HashSecret
	if secret == "" {
		secret = "anychat-verification-secret"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strings.Join([]string{purpose, target, code}, ":")))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *verifyServiceImpl) generateCode() string {
	if s.shouldUseDebugCode() {
		return s.config.DebugFixedCode
	}

	const codeChars = "0123456789"
	result := make([]byte, s.config.CodeLength)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(codeChars))))
		result[i] = codeChars[n.Int64()]
	}
	return string(result)
}

func (s *verifyServiceImpl) shouldUseDebugCode() bool {
	return !s.isReleaseMode() && s.config.DebugFixedCode != ""
}

func (s *verifyServiceImpl) shouldAllowDevBypass(code string) bool {
	return !s.isReleaseMode() && s.config.AllowDevBypass && s.config.DebugFixedCode != "" && code == s.config.DebugFixedCode
}

func (s *verifyServiceImpl) isReleaseMode() bool {
	return strings.EqualFold(s.config.AppMode, "release")
}

func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, n)
	for i := range result {
		pick, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		result[i] = letters[pick.Int64()]
	}
	return string(result)
}

func parseInt64(value string) (int64, error) {
	var result int64
	_, err := fmt.Sscanf(value, "%d", &result)
	return result, err
}

func maskTarget(target, targetType string) string {
	if target == "" {
		return ""
	}
	if targetType == model.TargetTypeEmail {
		parts := strings.SplitN(target, "@", 2)
		if len(parts) != 2 {
			return "***"
		}
		name := parts[0]
		if len(name) <= 2 {
			return "***@" + parts[1]
		}
		return name[:2] + "***@" + parts[1]
	}
	if len(target) <= 7 {
		return "***"
	}
	return target[:3] + "****" + target[len(target)-4:]
}

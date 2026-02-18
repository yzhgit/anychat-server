package jpush

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	pushURL = "https://api.jpush.cn/v3/push"
)

// Client 极光推送客户端
type Client struct {
	appKey       string
	masterSecret string
	httpClient   *http.Client
	auth         string // base64(appKey:masterSecret)
}

// NewClient 创建极光推送客户端
func NewClient(appKey, masterSecret string) *Client {
	auth := base64.StdEncoding.EncodeToString([]byte(appKey + ":" + masterSecret))
	return &Client{
		appKey:       appKey,
		masterSecret: masterSecret,
		auth:         auth,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// PushResult 推送结果
type PushResult struct {
	MsgID        string `json:"msg_id"`
	SendNo       string `json:"sendno"`
	SuccessCount int
	FailureCount int
}

// pushRequest JPush REST API v3 请求体
type pushRequest struct {
	Platform     interface{} `json:"platform"`
	Audience     audience    `json:"audience"`
	Notification *notification `json:"notification,omitempty"`
	Options      options     `json:"options"`
}

type audience struct {
	RegistrationID []string `json:"registration_id,omitempty"`
}

type notification struct {
	IOS     *iosNotification     `json:"ios,omitempty"`
	Android *androidNotification `json:"android,omitempty"`
}

type iosNotification struct {
	Alert  iosAlert          `json:"alert"`
	Sound  string            `json:"sound"`
	Badge  string            `json:"badge"`
	Extras map[string]string `json:"extras,omitempty"`
}

type iosAlert struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type androidNotification struct {
	Alert  string            `json:"alert"`
	Title  string            `json:"title"`
	Extras map[string]string `json:"extras,omitempty"`
}

type options struct {
	TimeToLive      int  `json:"time_to_live"`
	ApnsProduction  bool `json:"apns_production"`
}

type pushResponse struct {
	MsgID  string          `json:"msg_id"`
	SendNo string          `json:"sendno"`
	Error  *jpushErrorBody `json:"error,omitempty"`
}

type jpushErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// PushToRegistrationIDs 向指定 Registration ID 列表推送通知
// regIDs: 极光推送设备注册 ID（由 JPush SDK 生成）
func (c *Client) PushToRegistrationIDs(regIDs []string, title, content string, extras map[string]string) (*PushResult, error) {
	if len(regIDs) == 0 {
		return &PushResult{}, nil
	}

	req := pushRequest{
		Platform: "all",
		Audience: audience{RegistrationID: regIDs},
		Notification: &notification{
			IOS: &iosNotification{
				Alert: iosAlert{
					Title: title,
					Body:  content,
				},
				Sound:  "default",
				Badge:  "+1",
				Extras: extras,
			},
			Android: &androidNotification{
				Alert:  content,
				Title:  title,
				Extras: extras,
			},
		},
		Options: options{
			TimeToLive:     86400, // 1天内未收到则丢弃
			ApnsProduction: false, // 配置文件控制，此处默认开发环境
		},
	}

	result, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	result.SuccessCount = len(regIDs)
	return result, nil
}

func (c *Client) doRequest(payload interface{}) (*PushResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("jpush: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, pushURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("jpush: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+c.auth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jpush: http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("jpush: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jpush: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var pushResp pushResponse
	if err := json.Unmarshal(respBody, &pushResp); err != nil {
		return nil, fmt.Errorf("jpush: unmarshal response: %w", err)
	}

	if pushResp.Error != nil {
		return nil, fmt.Errorf("jpush: error %d: %s", pushResp.Error.Code, pushResp.Error.Message)
	}

	return &PushResult{
		MsgID:  pushResp.MsgID,
		SendNo: pushResp.SendNo,
	}, nil
}

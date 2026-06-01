package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const xizhiDefaultHost = "xizhi.qqoq.net"

func buildPushTitle(category string, templateName string) string {
	switch category {
	case categoryCombat:
		switch templateName {
		case "有人宣战":
			return "有人向你宣战"
		case "受到攻击":
			return "正在受到攻击"
		default:
			return "战斗提醒"
		}
	case categoryStatus:
		return "角色已死亡"
	case categoryNetwork:
		return "游戏可能已掉线"
	default:
		return "游戏监控提醒"
	}
}

func buildDisconnectPushContent(templateName string, score float64) string {
	host, _ := os.Hostname()
	if host == "" {
		host = "未知"
	}
	return fmt.Sprintf(
		"检测到画面「%s」\n相似度：%.2f\n时间：%s\n主机：%s",
		templateName,
		score,
		time.Now().Format("2006-01-02 15:04:05"),
		host,
	)
}

func buildRecoverPushContent() string {
	host, _ := os.Hostname()
	if host == "" {
		host = "未知"
	}
	return fmt.Sprintf(
		"掉线画面已消失，疑似已重连。\n时间：%s\n主机：%s",
		time.Now().Format("2006-01-02 15:04:05"),
		host,
	)
}

func buildXizhiPushURL(pushInput string, title string, content string) (string, error) {
	pushInput = strings.TrimSpace(pushInput)
	if pushInput == "" {
		return "", errors.New("推送链接/Key 不能为空")
	}

	title = strings.TrimSpace(title)
	if title == "" {
		title = "消息通知"
	}
	content = strings.TrimSpace(content)

	if strings.Contains(pushInput, "://") {
		u, err := url.Parse(pushInput)
		if err != nil {
			return "", err
		}
		if u.Scheme == "" || u.Host == "" {
			return "", errors.New("invalid push url")
		}
		q := u.Query()
		q.Set("title", title)
		q.Set("content", content)
		u.RawQuery = q.Encode()
		return u.String(), nil
	}

	key := strings.TrimSpace(pushInput)
	if strings.Contains(key, "/") {
		parts := strings.Split(key, "/")
		key = strings.TrimSpace(parts[len(parts)-1])
	}
	key = strings.TrimSuffix(key, ".send")
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("invalid push key")
	}

	u := &url.URL{
		Scheme: "https",
		Host:   xizhiDefaultHost,
		Path:   "/" + key + ".send",
	}
	q := u.Query()
	q.Set("title", title)
	q.Set("content", content)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func sendWechatPush(ctx context.Context, channelKey string, title string, content string) error {
	channelKey = strings.TrimSpace(channelKey)
	if channelKey == "" {
		return errors.New("推送链接/Key 不能为空")
	}

	pushURL, err := buildXizhiPushURL(channelKey, title, content)
	if err != nil {
		return err
	}

	pushClient := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pushURL, nil)
	if err != nil {
		return err
	}

	resp, err := pushClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return errors.New("HTTP " + resp.Status + ": " + strings.TrimSpace(string(respBody)))
	}

	return nil
}

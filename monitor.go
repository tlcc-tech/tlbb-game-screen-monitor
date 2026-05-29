package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type MonitorPhase string

const (
	phaseIdle           MonitorPhase = "idle"
	phaseWatching       MonitorPhase = "watching"
	phaseMatchedPending MonitorPhase = "matched_pending"
	phaseCooldown       MonitorPhase = "cooldown"
)

type MonitorStatus struct {
	Running           bool         `json:"running"`
	Phase             MonitorPhase `json:"phase"`
	LastScores        []MatchScore `json:"lastScores"`
	LastMatchedName   string       `json:"lastMatchedName"`
	LastMatchedScore  float64      `json:"lastMatchedScore"`
	LastChecked       string       `json:"lastChecked"`
	LastError         string       `json:"lastError"`
	NextPollIn        int          `json:"nextPollIn"`
	HitStreak         int          `json:"hitStreak"`
	PendingPush       bool         `json:"pendingPush"`
	CooldownRemaining int          `json:"cooldownRemaining"`
}

type Monitor struct {
	mu sync.Mutex

	appCtx context.Context

	running bool
	cancel  context.CancelFunc

	settings AppSettings
	matcher  Matcher

	hitStreak        int
	phase            MonitorPhase
	lastScores       []MatchScore
	lastMatchedName  string
	lastMatchedScore float64
	lastChecked      time.Time
	lastError        string
	nextPollAt       time.Time
	cooldownUntil    time.Time
	pendingTemplate  string
	pendingScore     float64
	wasMatched       bool
	pushInFlight     bool
	tplCache         map[string]image.Image
}

func NewMonitor() *Monitor {
	s, _ := loadSettings()
	return &Monitor{
		settings: s,
		matcher:  LookupMatcher{},
		phase:    phaseIdle,
		tplCache: make(map[string]image.Image),
	}
}

func (m *Monitor) Attach(appCtx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.appCtx = appCtx
	if s, err := loadSettings(); err == nil {
		m.settings = s
	}
}

func (m *Monitor) GetSettings() AppSettings {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.settings
}

func (m *Monitor) SaveSettings(s AppSettings) error {
	normalizeSettings(&s)
	m.mu.Lock()
	m.settings = s
	m.mu.Unlock()
	return saveSettings(s)
}

func (m *Monitor) Start(channelKey string, settings AppSettings) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return errors.New("监控已在运行")
	}

	normalizeSettings(&settings)
	settings.ChannelKey = strings.TrimSpace(channelKey)
	m.settings = settings
	if err := saveSettings(settings); err != nil {
		m.mu.Unlock()
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.running = true
	m.cancel = cancel
	m.phase = phaseWatching
	m.hitStreak = 0
	m.pendingTemplate = ""
	m.pendingScore = 0
	m.wasMatched = false
	m.lastError = ""
	m.mu.Unlock()

	go m.loop(ctx)
	m.emitLog("监控已启动")
	return nil
}

func (m *Monitor) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	if m.cancel != nil {
		m.cancel()
	}
	m.running = false
	m.phase = phaseIdle
	m.hitStreak = 0
	m.pendingTemplate = ""
	m.mu.Unlock()
	m.emitLog("监控已停止")
}

func (m *Monitor) Status() MonitorStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	nextIn := 0
	if m.running && !m.nextPollAt.IsZero() {
		nextIn = int(time.Until(m.nextPollAt).Seconds())
		if nextIn < 0 {
			nextIn = 0
		}
	}

	cooldownRem := 0
	if m.phase == phaseCooldown && !m.cooldownUntil.IsZero() {
		cooldownRem = int(time.Until(m.cooldownUntil).Seconds())
		if cooldownRem < 0 {
			cooldownRem = 0
		}
	}

	lastChecked := ""
	if !m.lastChecked.IsZero() {
		lastChecked = m.lastChecked.Format(time.RFC3339)
	}

	scores := make([]MatchScore, len(m.lastScores))
	copy(scores, m.lastScores)

	return MonitorStatus{
		Running:           m.running,
		Phase:             m.phase,
		LastScores:        scores,
		LastMatchedName:   m.lastMatchedName,
		LastMatchedScore:  m.lastMatchedScore,
		LastChecked:       lastChecked,
		LastError:         m.lastError,
		NextPollIn:        nextIn,
		HitStreak:         m.hitStreak,
		PendingPush:       m.phase == phaseMatchedPending,
		CooldownRemaining: cooldownRem,
	}
}

func (m *Monitor) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		m.mu.Lock()
		interval := time.Duration(m.settings.PollIntervalSec) * time.Second
		if m.phase == phaseCooldown {
			if time.Now().Before(m.cooldownUntil) {
				wait := time.Until(m.cooldownUntil)
				m.nextPollAt = time.Now().Add(wait)
				m.mu.Unlock()
				select {
				case <-ctx.Done():
					return
				case <-time.After(wait):
				}
				continue
			}
			m.phase = phaseWatching
			m.hitStreak = 0
		}
		settings := m.settings
		m.mu.Unlock()

		m.checkOnce(ctx, settings)

		m.mu.Lock()
		m.nextPollAt = time.Now().Add(interval)
		m.mu.Unlock()

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func (m *Monitor) checkOnce(ctx context.Context, settings AppSettings) {
	m.mu.Lock()
	if m.phase == phaseMatchedPending {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	start := time.Now()
	m.emitLog("开始识别…")

	screen, err := captureScreen(settings.GameWindowTitle)
	if err != nil {
		m.setError(err.Error())
		m.emitLog("截图失败: " + err.Error())
		return
	}

	scores := m.matchAll(screen)
	elapsed := time.Since(start)
	m.mu.Lock()
	m.lastScores = scores
	m.lastChecked = time.Now()
	m.lastError = ""
	m.mu.Unlock()

	m.emitMatch(scores)
	m.emitLog(fmt.Sprintf("识别完成，耗时 %.1fs", elapsed.Seconds()))

	bestName := ""
	bestScore := 0.0
	matched := false
	for _, s := range scores {
		if s.Matched && s.Score > bestScore {
			matched = true
			bestScore = s.Score
			bestName = s.TemplateName
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !matched {
		if m.wasMatched && settings.NotifyOnRecover {
			m.tryRecoverNotify(ctx, settings)
		}
		m.hitStreak = 0
		m.wasMatched = false
		return
	}

	m.hitStreak++
	m.lastMatchedName = bestName
	m.lastMatchedScore = bestScore
	m.wasMatched = true

	if m.hitStreak < settings.ConsecutiveHits {
		m.emitLog(fmt.Sprintf("命中 %d/%d：%s (%.3f)", m.hitStreak, settings.ConsecutiveHits, bestName, bestScore))
		return
	}

	m.phase = phaseMatchedPending
	m.pendingTemplate = bestName
	m.pendingScore = bestScore
	if !m.pushInFlight {
		m.pushInFlight = true
		m.emitLog(fmt.Sprintf("连续命中，等待网络恢复后推送：%s (%.3f)", bestName, bestScore))
		go m.handlePendingPush(ctx, settings, bestName, bestScore)
	}
}

func (m *Monitor) handlePendingPush(ctx context.Context, settings AppSettings, templateName string, score float64) {
	defer func() {
		m.mu.Lock()
		m.pushInFlight = false
		m.mu.Unlock()
	}()

	deadline := time.Now().Add(time.Duration(settings.NetworkWaitMaxMin) * time.Minute)
	probeInterval := 3 * time.Second
	prober := NewNetworkProber(settings)

	if prober.IsOnline(ctx) {
		m.emitLog("网络已连通，立即推送")
	} else {
		m.emitLog("等待网络恢复…")
		waitCtx, cancel := context.WithDeadline(ctx, deadline)
		recovered := waitForNetwork(waitCtx, settings, 2, probeInterval)
		cancel()
		if !recovered {
			select {
			case <-ctx.Done():
				return
			default:
				m.emitLog("网络等待超时，仍将尝试推送")
			}
		}
	}

	channelKey := strings.TrimSpace(settings.ChannelKey)
	if channelKey == "" {
		m.emitLog("未配置推送链接，跳过微信推送")
		m.enterCooldown(settings)
		return
	}

	title := "游戏可能已掉线"
	content := buildDisconnectPushContent(templateName, score)
	if err := sendWechatPush(ctx, channelKey, title, content); err != nil {
		m.setError("推送失败: " + err.Error())
		m.emitLog("推送失败: " + err.Error())
		return
	}

	m.emitLog("微信推送成功")
	m.enterCooldown(settings)
}

func (m *Monitor) enterCooldown(settings AppSettings) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.phase = phaseCooldown
	m.cooldownUntil = time.Now().Add(time.Duration(settings.PushCooldownMin) * time.Minute)
	m.hitStreak = 0
	m.pendingTemplate = ""
	m.wasMatched = false
}

func (m *Monitor) tryRecoverNotify(ctx context.Context, settings AppSettings) {
	if strings.TrimSpace(settings.ChannelKey) == "" {
		return
	}
	if !NewNetworkProber(settings).IsOnline(ctx) {
		return
	}
	title := "游戏疑似已重连"
	content := buildRecoverPushContent()
	if err := sendWechatPush(ctx, settings.ChannelKey, title, content); err != nil {
		m.emitLog("恢复通知失败: " + err.Error())
		return
	}
	m.emitLog("已发送重连通知")
}

func (m *Monitor) setError(msg string) {
	m.mu.Lock()
	m.lastError = msg
	m.mu.Unlock()
}

func (m *Monitor) emitLog(line string) {
	if m.appCtx == nil {
		return
	}
	runtime.EventsEmit(m.appCtx, "log", time.Now().Format("15:04:05")+" "+line)
}

func (m *Monitor) emitMatch(scores []MatchScore) {
	if m.appCtx == nil {
		return
	}
	runtime.EventsEmit(m.appCtx, "match", map[string]interface{}{
		"scores": scores,
		"at":     time.Now().Format(time.RFC3339),
	})
}

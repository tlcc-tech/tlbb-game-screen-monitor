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
)

type TemplatePushCooldown struct {
	TemplateID   string `json:"templateId"`
	TemplateName string `json:"templateName"`
	RemainingSec int    `json:"remainingSec"`
}

type MonitorStatus struct {
	Running           bool                   `json:"running"`
	Phase             MonitorPhase           `json:"phase"`
	LastScores        []MatchScore           `json:"lastScores"`
	LastMatchedName   string                 `json:"lastMatchedName"`
	LastMatchedScore  float64                `json:"lastMatchedScore"`
	LastChecked       string                 `json:"lastChecked"`
	LastError         string                 `json:"lastError"`
	NextPollIn        int                    `json:"nextPollIn"`
	PendingPush       bool                   `json:"pendingPush"`
	PendingTemplates  []string               `json:"pendingTemplates"`
	PushCooldowns     []TemplatePushCooldown `json:"pushCooldowns"`
}

type Monitor struct {
	mu sync.Mutex

	appCtx context.Context

	running bool
	cancel  context.CancelFunc

	settings AppSettings
	matcher  Matcher

	phase         MonitorPhase
	lastScores    []MatchScore
	lastMatchedName  string
	lastMatchedScore float64
	lastChecked   time.Time
	lastError     string
	nextPollAt    time.Time

	hitStreaks        map[string]int
	pushCooldownUntil map[string]time.Time
	pendingPush       map[string]bool
	wasMatched        map[string]bool

	tplCache map[string]image.Image
}

func NewMonitor() *Monitor {
	s, _ := loadSettings()
	return &Monitor{
		settings:          s,
		matcher:           LookupMatcher{},
		phase:             phaseIdle,
		tplCache:          make(map[string]image.Image),
		hitStreaks:        make(map[string]int),
		pushCooldownUntil: make(map[string]time.Time),
		pendingPush:       make(map[string]bool),
		wasMatched:        make(map[string]bool),
	}
}

func (m *Monitor) resetTemplateStateLocked() {
	m.hitStreaks = make(map[string]int)
	m.pushCooldownUntil = make(map[string]time.Time)
	m.pendingPush = make(map[string]bool)
	m.wasMatched = make(map[string]bool)
}

func (m *Monitor) updatePhaseLocked() {
	if len(m.pendingPush) > 0 {
		m.phase = phaseMatchedPending
		return
	}
	if m.running {
		m.phase = phaseWatching
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
	m.resetTemplateStateLocked()
	m.lastError = ""
	m.mu.Unlock()

	go m.loop(ctx)
	if settings.GameWindowHwnd != 0 {
		m.emitLog(fmt.Sprintf("已绑定窗口: %s (0x%X)", settings.GameWindowTitle, settings.GameWindowHwnd))
	}
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
	m.resetTemplateStateLocked()
	m.mu.Unlock()
	releaseDmCaptureBinding()
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

	lastChecked := ""
	if !m.lastChecked.IsZero() {
		lastChecked = m.lastChecked.Format(time.RFC3339)
	}

	scores := make([]MatchScore, len(m.lastScores))
	copy(scores, m.lastScores)

	var pushCooldowns []TemplatePushCooldown
	nameByID := make(map[string]string, len(m.settings.Templates))
	for _, t := range m.settings.Templates {
		nameByID[t.ID] = t.Name
	}
	for id, until := range m.pushCooldownUntil {
		rem := int(time.Until(until).Seconds())
		if rem <= 0 {
			continue
		}
		name := nameByID[id]
		if name == "" {
			name = id
		}
		pushCooldowns = append(pushCooldowns, TemplatePushCooldown{
			TemplateID:   id,
			TemplateName: name,
			RemainingSec: rem,
		})
	}

	pendingTemplates := make([]string, 0, len(m.pendingPush))
	for id := range m.pendingPush {
		name := nameByID[id]
		if name == "" {
			name = id
		}
		pendingTemplates = append(pendingTemplates, name)
	}

	return MonitorStatus{
		Running:          m.running,
		Phase:            m.phase,
		LastScores:       scores,
		LastMatchedName:  m.lastMatchedName,
		LastMatchedScore: m.lastMatchedScore,
		LastChecked:      lastChecked,
		LastError:        m.lastError,
		NextPollIn:       nextIn,
		PendingPush:      len(m.pendingPush) > 0,
		PendingTemplates: pendingTemplates,
		PushCooldowns:    pushCooldowns,
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
	start := time.Now()
	m.emitLog("开始识别…")

	capStart := time.Now()
	screen, err := captureScreen(settings)
	capElapsed := time.Since(capStart)
	if err != nil {
		m.setError(err.Error())
		m.emitLog("截图失败: " + err.Error())
		return
	}

	matchStart := time.Now()
	scores := m.matchAll(screen)
	matchElapsed := time.Since(matchStart)
	elapsed := time.Since(start)

	m.mu.Lock()
	m.lastScores = scores
	m.lastChecked = time.Now()
	m.lastError = ""
	m.mu.Unlock()

	m.emitMatch(scores)
	m.emitLog(fmt.Sprintf("识别完成，截图 %.1fs / 匹配 %.1fs（共 %.1fs）", capElapsed.Seconds(), matchElapsed.Seconds(), elapsed.Seconds()))

	bestName := ""
	bestScore := 0.0
	matchedAny := false
	for _, s := range scores {
		if s.Matched {
			matchedAny = true
			if s.Score > bestScore {
				bestScore = s.Score
				bestName = s.TemplateName
			}
			m.processMatch(ctx, settings, s)
		} else {
			m.processMiss(ctx, settings, s)
		}
	}

	m.mu.Lock()
	if matchedAny {
		m.lastMatchedName = bestName
		m.lastMatchedScore = bestScore
	}
	m.mu.Unlock()
}

func (m *Monitor) processMiss(ctx context.Context, settings AppSettings, s MatchScore) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wasMatched[s.TemplateID] && settings.NotifyOnRecover && s.Category == categoryNetwork {
		go m.tryRecoverNotify(ctx, settings)
	}
	delete(m.hitStreaks, s.TemplateID)
	m.wasMatched[s.TemplateID] = false
}

func (m *Monitor) processMatch(ctx context.Context, settings AppSettings, s MatchScore) {
	m.mu.Lock()

	if until, ok := m.pushCooldownUntil[s.TemplateID]; ok && time.Now().Before(until) {
		rem := int(time.Until(until).Seconds())
		name := s.TemplateName
		m.wasMatched[s.TemplateID] = true
		m.mu.Unlock()
		m.emitLog(fmt.Sprintf("推送冷却中，跳过：%s（剩余 %ds）", name, rem))
		return
	}

	if m.pendingPush[s.TemplateID] {
		m.wasMatched[s.TemplateID] = true
		m.mu.Unlock()
		return
	}

	m.wasMatched[s.TemplateID] = true
	m.hitStreaks[s.TemplateID]++
	streak := m.hitStreaks[s.TemplateID]
	templateName := s.TemplateName
	score := s.Score

	if streak < settings.ConsecutiveHits {
		m.mu.Unlock()
		m.emitLog(fmt.Sprintf("命中 %d/%d：%s (%.3f)", streak, settings.ConsecutiveHits, templateName, score))
		return
	}

	m.pendingPush[s.TemplateID] = true
	m.updatePhaseLocked()
	category := s.Category
	templateID := s.TemplateID
	m.mu.Unlock()

	if category == categoryNetwork {
		m.emitLog(fmt.Sprintf("连续命中，等待网络恢复后推送：%s (%.3f)", templateName, score))
		go m.runPendingPush(ctx, settings, templateID, templateName, score, category)
	} else {
		m.emitLog(fmt.Sprintf("命中，立即推送：%s (%.3f)", templateName, score))
		go m.runInstantPush(ctx, settings, templateID, templateName, score, category)
	}
}

func (m *Monitor) clearPendingPush(templateID string) {
	m.mu.Lock()
	delete(m.pendingPush, templateID)
	m.updatePhaseLocked()
	m.mu.Unlock()
}

func (m *Monitor) runInstantPush(ctx context.Context, settings AppSettings, templateID, templateName string, score float64, category string) {
	defer m.clearPendingPush(templateID)
	m.handleInstantPush(ctx, settings, templateID, templateName, score, category)
}

func (m *Monitor) runPendingPush(ctx context.Context, settings AppSettings, templateID, templateName string, score float64, category string) {
	defer m.clearPendingPush(templateID)
	m.handlePendingPush(ctx, settings, templateID, templateName, score, category)
}

func (m *Monitor) handleInstantPush(ctx context.Context, settings AppSettings, templateID, templateName string, score float64, category string) {
	m.deliverAlert(ctx, settings, templateID, category, templateName, score)
}

func (m *Monitor) handlePendingPush(ctx context.Context, settings AppSettings, templateID, templateName string, score float64, category string) {
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

	m.deliverAlert(ctx, settings, templateID, category, templateName, score)
}

func (m *Monitor) deliverAlert(ctx context.Context, settings AppSettings, templateID, category, templateName string, score float64) {
	title := buildPushTitle(category, templateName)
	content := buildDisconnectPushContent(templateName, score)

	if err := showDesktopNotification(title, content); err != nil {
		m.emitLog("桌面通知失败: " + err.Error())
	} else {
		m.emitLog("已发送桌面通知")
	}

	channelKey := strings.TrimSpace(settings.ChannelKey)
	if channelKey == "" {
		m.emitLog("未配置推送链接，仅桌面通知")
	} else if err := sendWechatPush(ctx, channelKey, title, content); err != nil {
		m.setError("推送失败: " + err.Error())
		m.emitLog("推送失败: " + err.Error())
	} else {
		m.emitLog("微信推送成功")
	}

	m.setPushCooldown(templateID, settings)
}

func (m *Monitor) setPushCooldown(templateID string, settings AppSettings) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pushCooldownUntil[templateID] = time.Now().Add(time.Duration(settings.PushCooldownMin) * time.Minute)
	delete(m.hitStreaks, templateID)
}

func (m *Monitor) tryRecoverNotify(ctx context.Context, settings AppSettings) {
	if !NewNetworkProber(settings).IsOnline(ctx) {
		return
	}

	title := "游戏疑似已重连"
	content := buildRecoverPushContent()

	if err := showDesktopNotification(title, content); err != nil {
		m.emitLog("桌面通知失败: " + err.Error())
	}

	channelKey := strings.TrimSpace(settings.ChannelKey)
	if channelKey == "" {
		return
	}
	if err := sendWechatPush(ctx, channelKey, title, content); err != nil {
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

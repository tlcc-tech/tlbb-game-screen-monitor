package main

import (
	"context"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type NetworkProber struct {
	settings AppSettings
	client   *http.Client
}

func NewNetworkProber(settings AppSettings) *NetworkProber {
	return &NetworkProber{
		settings: settings,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (p *NetworkProber) IsOnline(ctx context.Context) bool {
	okPing := true
	okHTTP := true

	if p.settings.UsePing {
		okPing = p.ping(ctx, strings.TrimSpace(p.settings.PingHost))
	}
	if p.settings.UseHttp {
		okHTTP = p.httpHead(ctx, strings.TrimSpace(p.settings.HttpProbeURL))
	}

	if p.settings.UsePing && p.settings.UseHttp {
		return okPing && okHTTP
	}
	if p.settings.UsePing {
		return okPing
	}
	if p.settings.UseHttp {
		return okHTTP
	}
	return true
}

func (p *NetworkProber) ping(ctx context.Context, host string) bool {
	if host == "" {
		return false
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "-w", "1000", host)
	} else {
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", host)
	}
	return cmd.Run() == nil
}

func (p *NetworkProber) httpHead(ctx context.Context, rawURL string) bool {
	if rawURL == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return false
	}
	resp, err := p.client.Do(req)
	if err != nil {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return false
		}
		resp, err = p.client.Do(req)
		if err != nil {
			return false
		}
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

func waitForNetwork(ctx context.Context, settings AppSettings, requiredSuccess int, interval time.Duration) bool {
	if requiredSuccess < 1 {
		requiredSuccess = 2
	}
	prober := NewNetworkProber(settings)
	streak := 0
	for {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		if prober.IsOnline(ctx) {
			streak++
			if streak >= requiredSuccess {
				return true
			}
		} else {
			streak = 0
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(interval):
		}
	}
}

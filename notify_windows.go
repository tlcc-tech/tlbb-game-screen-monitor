//go:build windows

package main

import (
	"strings"

	"github.com/go-toast/toast"
)

func showDesktopNotification(title, message string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		title = AppName
	}
	n := toast.Notification{
		AppID:   AppName,
		Title:   title,
		Message: strings.TrimSpace(message),
	}
	return n.Push()
}

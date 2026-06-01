//go:build !windows

package main

func setupHotkeys(_ *App, _ *Monitor) {}
func refreshHotkeys(_ *Monitor)       {}
func stopHotkeys()                    {}

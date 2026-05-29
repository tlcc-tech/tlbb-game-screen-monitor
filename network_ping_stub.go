//go:build !windows

package main

import "os/exec"

func hidePingCmd(cmd *exec.Cmd) {}

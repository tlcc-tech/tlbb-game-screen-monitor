//go:build windows

package main

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func initRuntimeDLLPaths() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	base := filepath.Join(filepath.Dir(exe), "runtime")
	dirs := []string{
		filepath.Join(base, "opencv"),
		filepath.Join(base, "mingw"),
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	addDllDirectory := kernel32.NewProc("AddDllDirectory")
	setDefaultDllDirectories := kernel32.NewProc("SetDefaultDllDirectories")
	const loadLibrarySearchDefaultDirs = 0x00001000
	const loadLibrarySearchUserDirs = 0x00000400

	setDefaultDllDirectories.Call(loadLibrarySearchDefaultDirs | loadLibrarySearchUserDirs)
	for _, dir := range dirs {
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			continue
		}
		ptr, err := syscall.UTF16PtrFromString(dir)
		if err != nil {
			continue
		}
		addDllDirectory.Call(uintptr(unsafe.Pointer(ptr)))
	}
}

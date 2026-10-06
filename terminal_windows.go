package main

import (
	"fmt"
	"os"
	"syscall"
)

// Windows consoles need virtual-terminal processing enabled for ANSI output.
func prepareTerminal() (func(), error) {
	handle := syscall.Handle(os.Stdout.Fd())
	var mode uint32
	if err := syscall.GetConsoleMode(handle, &mode); err != nil {
		return nil, fmt.Errorf("read console mode: %w (try --once)", err)
	}
	setMode := syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode")
	const enableVirtualTerminalProcessing = 0x0004
	ok, _, err := setMode.Call(uintptr(handle), uintptr(mode|enableVirtualTerminalProcessing))
	if ok == 0 {
		return nil, fmt.Errorf("enable terminal output: %w (try Windows Terminal or --once)", err)
	}
	return func() { _, _, _ = setMode.Call(uintptr(handle), uintptr(mode)) }, nil
}

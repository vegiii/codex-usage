//go:build !windows

package main

func prepareTerminal() (func(), error) {
	return func() {}, nil
}

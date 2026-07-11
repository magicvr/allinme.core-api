//go:build windows

package applock

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

type Lock struct {
	handle windows.Handle
	path   string
}

func Acquire(path string) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("encode lock path: %w", err)
	}
	handle, err := windows.CreateFile(pathPointer, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, fmt.Errorf("acquire API process lock: %w", err)
	}
	return &Lock{handle: handle, path: path}, nil
}

func (lock *Lock) Close() error {
	if lock == nil || lock.handle == windows.InvalidHandle {
		return nil
	}
	err := windows.CloseHandle(lock.handle)
	lock.handle = windows.InvalidHandle
	if removeErr := os.Remove(lock.path); removeErr != nil && !os.IsNotExist(removeErr) && err == nil {
		err = removeErr
	}
	return err
}

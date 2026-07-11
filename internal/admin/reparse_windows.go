//go:build windows

package admin

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func rejectReparsePoint(path string) error {
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("inspect reset path: %w", err)
	}
	attributes, err := windows.GetFileAttributes(pathPointer)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect reset path: %w", err)
	}
	if attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fmt.Errorf("reset path must not be a reparse point")
	}
	return nil
}

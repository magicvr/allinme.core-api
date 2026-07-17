//go:build windows

package files

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func rejectAttachmentRoot(path string) error {
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return safeFileError("inspect attachment storage", err)
	}
	attributes, err := windows.GetFileAttributes(pathPointer)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return safeFileError("inspect attachment storage", err)
	}
	if attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fmt.Errorf("initialize attachment storage: root must not be a reparse point")
	}
	return nil
}

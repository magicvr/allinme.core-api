//go:build !windows

package files

import (
	"fmt"
	"os"
)

func rejectAttachmentRoot(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return safeFileError("inspect attachment storage", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("initialize attachment storage: root must not be a symbolic link")
	}
	return nil
}

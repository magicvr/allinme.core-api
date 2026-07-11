//go:build !windows

package admin

import (
	"fmt"
	"os"
)

func rejectReparsePoint(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect reset path: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("reset path must not be a symbolic link")
	}
	return nil
}

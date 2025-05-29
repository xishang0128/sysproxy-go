//go:build !darwin

package sysproxy

import (
	"fmt"
	"runtime"
)

func Start(_ string) error {
	return fmt.Errorf("未支持%s", runtime.GOOS)
}

//go:build !darwin && !linux && !windows

package sysproxy

import "fmt"

func DisableProxy(_ string, _ bool) error {
	return fmt.Errorf("不支持的操作系统")
}

func SetProxy(proxy, bypass, _ string, _ bool) error {
	return fmt.Errorf("不支持的操作系统")
}

func SetPac(server, _ string, _ bool) error {
	return fmt.Errorf("不支持的操作系统")
}

func QueryProxySettings(_ string, _ bool) (map[string]any, error) {
	return nil, fmt.Errorf("不支持的操作系统")
}

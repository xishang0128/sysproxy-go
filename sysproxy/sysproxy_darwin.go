//go:build darwin

package sysproxy

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

func DisableProxy() error {
	services, err := getNetworkServices()
	if err != nil {
		return err
	}

	commands := [][]string{
		{"-setautoproxystate", "off"},
		{"-setproxyautodiscovery", "off"},
		{"-setwebproxystate", "off"},
		{"-setsecurewebproxystate", "off"},
		{"-setsocksfirewallproxystate", "off"},
	}

	errChan := make(chan error, len(services))
	var wg sync.WaitGroup

	for _, service := range services {
		wg.Add(1)
		go func(svc string) {
			defer wg.Done()
			if err := execNetworksetupConcurrent(svc, commands); err != nil {
				errChan <- err
			}
		}(service)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func SetProxy(proxy, bypass string) error {
	if proxy == "" || bypass == "" {
		config, err := QueryProxySettings()
		if err != nil {
			return err
		}

		if proxy == "" {
			proxy = config.Proxy.Servers["http_server"]
		}
		if bypass == "" {
			bypass = config.Proxy.Bypass
		}
	}

	addr := ParseServerString(proxy)
	if addr.host == "" || addr.port == "" {
		return fmt.Errorf("invalid proxy address: %s", proxy)
	}

	services, err := getNetworkServices()
	if err != nil {
		return err
	}

	commands := [][]string{
		{"-setautoproxystate", "off"},
		{"-setproxyautodiscovery", "off"},
		{"-setwebproxy", addr.host, addr.port},
		{"-setsecurewebproxy", addr.host, addr.port},
		{"-setsocksfirewallproxy", addr.host, addr.port},
		{"-setproxybypassdomains", bypass},
	}

	errChan := make(chan error, len(services))
	var wg sync.WaitGroup

	for _, service := range services {
		wg.Add(1)
		go func(svc string) {
			defer wg.Done()
			if err := execNetworksetupConcurrent(svc, commands); err != nil {
				errChan <- err
			}
		}(service)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func SetPac(pacUrl string) error {
	if pacUrl == "" {
		config, err := QueryProxySettings()
		if err != nil {
			return err
		}
		pacUrl = config.PAC.URL
	}

	services, err := getNetworkServices()
	if err != nil {
		return err
	}

	commands := [][]string{
		{"-setwebproxystate", "off"},
		{"-setsecurewebproxystate", "off"},
		{"-setsocksfirewallproxystate", "off"},
		{"-setautoproxyurl", pacUrl},
		{"-setautoproxystate", "on"},
		{"-setproxyautodiscovery", "on"},
	}

	errChan := make(chan error, len(services))
	var wg sync.WaitGroup

	for _, service := range services {
		wg.Add(1)
		go func(svc string) {
			defer wg.Done()
			if err := execNetworksetupConcurrent(svc, commands); err != nil {
				errChan <- err
			}
		}(service)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func QueryProxySettings() (*ProxyConfig, error) {
	services, err := getNetworkServices()
	if err != nil {
		return nil, err
	}

	service := services[0]
	config := &ProxyConfig{}

	output, err := exec.Command("networksetup", "-getautoproxyurl", service).Output()
	if err == nil && strings.Contains(string(output), "Enabled: Yes") {
		config.PAC.Enable = true
		lines := strings.SplitSeq(string(output), "\n")
		for line := range lines {
			if strings.HasPrefix(line, "URL: ") {
				config.PAC.URL = strings.TrimPrefix(line, "URL: ")
				break
			}
		}
	}

	output, err = exec.Command("networksetup", "-getwebproxy", service).Output()
	if err == nil && strings.Contains(string(output), "Enabled: Yes") {
		config.Proxy.Enable = true
		lines := strings.SplitSeq(string(output), "\n")
		if config.Proxy.Servers == nil {
			config.Proxy.Servers = make(map[string]string)
		}
		var host, port string
		for line := range lines {
			if strings.HasPrefix(line, "Server: ") {
				host = strings.TrimPrefix(line, "Server: ")
			} else if strings.HasPrefix(line, "Port: ") {
				port = strings.TrimPrefix(line, "Port: ")
			}
		}
		config.Proxy.Servers["http_server"] = FormatServer(host, port)
	}

	output, err = exec.Command("networksetup", "-getproxybypassdomains", service).Output()
	if err == nil {
		config.Proxy.Bypass = strings.TrimSpace(string(output))
	}

	return config, nil
}

func getNetworkServices() ([]string, error) {
	output, err := exec.Command("networksetup", "-listnetworkserviceorder").Output()
	if err != nil {
		return nil, fmt.Errorf("无法执行 networksetup 命令: %s", err)
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("networksetup 命令没有输出")
	}

	var services []string
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		if len(line) == 0 {
			continue
		}

		if strings.HasPrefix(line, "(") {
			parts := strings.Split(line, ")")
			if len(parts) < 2 {
				continue
			}
			service := strings.TrimSpace(parts[1])
			if service != "" {
				services = append(services, service)
			}
		}
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("未找到网络服务")
	}

	return services, nil
}

func execNetworksetupConcurrent(service string, commands [][]string) error {
	errChan := make(chan error, len(commands))
	var wg sync.WaitGroup

	for _, cmd := range commands {
		wg.Add(1)
		go func(args []string) {
			defer wg.Done()
			if err := exec.Command("networksetup", args...).Run(); err != nil {
				errChan <- fmt.Errorf("执行 networksetup %v 时出错，服务 %s: %w", args, service, err)
			}
		}(append([]string{cmd[0]}, append([]string{service}, cmd[1:]...)...))
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

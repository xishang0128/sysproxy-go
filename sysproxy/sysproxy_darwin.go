//go:build darwin

package sysproxy

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

func DisableProxy(device string, onlyWithDevice bool) error {
	var (
		services []string
		err      error
	)
	if device != "" {
		services = []string{device}
	} else {
		services, err = getNetworkServices(onlyWithDevice)
		if err != nil {
			return err
		}
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

func SetProxy(proxy, bypass, device string, onlyWithDevice bool) error {
	if proxy == "" || bypass == "" {
		config, err := QueryProxySettings(device, onlyWithDevice)
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

	var (
		services []string
		err      error
	)
	if device != "" {
		services = []string{device}
	} else {
		services, err = getNetworkServices(onlyWithDevice)
		if err != nil {
			return err
		}
	}

	commands := [][]string{
		{"-setautoproxystate", "off"},
		{"-setproxyautodiscovery", "off"},
		{"-setwebproxy", addr.host, addr.port},
		{"-setsecurewebproxy", addr.host, addr.port},
		{"-setsocksfirewallproxy", addr.host, addr.port},
		append([]string{"-setproxybypassdomains"}, strings.Split(bypass, ",")...),
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

func SetPac(pacUrl, device string, onlyWithDevice bool) error {
	if pacUrl == "" {
		config, err := QueryProxySettings(device, onlyWithDevice)
		if err != nil {
			return err
		}
		pacUrl = config.PAC.URL
	}

	var (
		services []string
		err      error
	)
	if device != "" {
		services = []string{device}
	} else {
		services, err = getNetworkServices(onlyWithDevice)
		if err != nil {
			return err
		}
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

func QueryProxySettings(device string, onlyWithDevice bool) (*ProxyConfig, error) {
	var (
		services []string
		err      error
	)
	if device != "" {
		services = []string{device}
	} else {
		services, err = getNetworkServices(onlyWithDevice)
		if err != nil {
			return nil, err
		}
	}

	service := services[0]
	config := &ProxyConfig{}
	config.Proxy.Servers = make(map[string]string)

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

	if enabled, host, port := parseProxy(exec.Command("networksetup", "-getwebproxy", service)); enabled {
		config.Proxy.Enable = true
		if addr := FormatServer(host, port); addr != "" {
			config.Proxy.Servers["http_server"] = addr
		}
	}

	if enabled, host, port := parseProxy(exec.Command("networksetup", "-getsecurewebproxy", service)); enabled {
		config.Proxy.Enable = true
		if addr := FormatServer(host, port); addr != "" {
			config.Proxy.Servers["https_server"] = addr
		}
	}

	if enabled, host, port := parseProxy(exec.Command("networksetup", "-getsocksfirewallproxy", service)); enabled {
		config.Proxy.Enable = true
		if addr := FormatServer(host, port); addr != "" {
			config.Proxy.Servers["socks_server"] = addr
		}
	}

	if output, err := exec.Command("networksetup", "-getproxybypassdomains", service).Output(); err == nil {
		bypass := strings.ReplaceAll(strings.TrimSpace(string(output)), "\n", ",")
		if bypass != "" {
			config.Proxy.Bypass = bypass
		}
	}

	return config, nil
}

func getNetworkServices(onlyWithDevice bool) ([]string, error) {
	cmd := exec.Command("networksetup", "-listnetworkserviceorder")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("无法执行 networksetup 命令: %w", err)
	}
	if len(output) == 0 {
		return nil, fmt.Errorf("networksetup 命令没有输出")
	}

	ordinalRegex := regexp.MustCompile(`^\(\d+\)\s*(.+)$`)
	deviceRegex := regexp.MustCompile(`Device:\s*([^\s,)]+)`)

	var services []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if m := ordinalRegex.FindStringSubmatch(line); m != nil {
			service := strings.TrimSpace(m[1])

			device := ""
			if scanner.Scan() {
				next := scanner.Text()
				if dm := deviceRegex.FindStringSubmatch(next); dm != nil {
					device = dm[1]
				}
			}
			if onlyWithDevice && device == "" {
				continue
			}
			services = append(services, service)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("扫描输出时出错: %w", err)
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

func parseProxy(cmd *exec.Cmd) (enabled bool, host, port string) {
	if output, err := cmd.Output(); err == nil {
		for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
			switch {
			case strings.HasPrefix(line, "Enabled: Yes"):
				enabled = true
			case strings.HasPrefix(line, "Server: "):
				host = strings.TrimPrefix(line, "Server: ")
			case strings.HasPrefix(line, "Port: "):
				port = strings.TrimPrefix(line, "Port: ")
			}
		}
	}
	return
}

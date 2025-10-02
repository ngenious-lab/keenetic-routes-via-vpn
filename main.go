package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	routesFilePath = "/opt/etc/vpn-router/current_routes.txt"
	rulePriority   = "1995"
	routeTable     = "1000"
)

// Config описывает конфигурацию
type Config struct {
	VPNInterface string   `yaml:"vpn_interface"`
	RepoDir      string   `yaml:"repo_dir"`
	Files        []string `yaml:"files"`
	IPs          []string `yaml:"ips"`
}

// loadConfig загружает конфигурацию
func loadConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("чтение config.yaml: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("разбор config.yaml: %w", err)
	}
	return cfg, nil
}

// parseRoutes собирает маршруты из .bat и конфигурации
func parseRoutes(cfg Config) []string {
	var routes []string

	for _, file := range cfg.Files {
		path := filepath.Join(cfg.RepoDir, file)
		data, err := os.ReadFile(path)
		if err != nil {
			logWarn("не удалось прочитать %s: %v", path, err)
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "route ADD ") {
				parts := strings.Fields(line)
				if len(parts) >= 5 {
					if cidr, err := maskToCIDR(parts[2], parts[4]); err == nil {
						routes = append(routes, cidr)
					} else {
						logWarn("неверный маршрут %s: %v", line, err)
					}
				}
			}
		}
	}

	for _, ip := range cfg.IPs {
		if isValidCIDR(ip) {
			routes = append(routes, ip)
		} else {
			logWarn("неверный формат CIDR %s, пропускаем", ip)
		}
	}

	return removeDuplicates(routes)
}

// applyRoutes пишет маршруты в файл и применяет в таблицу
func applyRoutes(cfg Config, routes []string) error {
	if err := os.WriteFile(routesFilePath, []byte(strings.Join(routes, "\n")+"\n"), 0644); err != nil {
		return fmt.Errorf("запись %s: %w", routesFilePath, err)
	}
	logInfo("маршруты сохранены в %s", routesFilePath)

	if !isInterfaceUp(cfg.VPNInterface) {
		logWarn("VPN-интерфейс %s не активен, пропускаем применение маршрутов", cfg.VPNInterface)
		return nil
	}

	if err := resetRouting(); err != nil {
		logWarn("сброс таблицы/правил: %v", err)
	}

	if err := addRule(); err != nil {
		return fmt.Errorf("добавление правила: %w", err)
	}

	data, err := os.ReadFile(routesFilePath)
	if err != nil {
		return fmt.Errorf("чтение %s: %w", routesFilePath, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !isValidCIDR(line) {
			continue
		}
		if err := exec.Command("ip", "route", "add", "table", routeTable, line, "dev", cfg.VPNInterface).Run(); err != nil {
			logWarn("не удалось добавить маршрут %s: %v", line, err)
		}
	}

	logInfo("маршруты применены в таблицу %s", routeTable)
	return nil
}

// resetRouting очищает таблицу и удаляет правило
func resetRouting() error {
	if err := exec.Command("ip", "route", "flush", "table", routeTable).Run(); err != nil {
		return err
	}
	// удаляем правило если существует
	if out, err := exec.Command("ip", "rule", "show").CombinedOutput(); err == nil {
		if strings.Contains(string(out), rulePriority) {
			_ = exec.Command("ip", "rule", "del", "priority", rulePriority).Run()
		}
	}
	return nil
}

func addRule() error {
	return exec.Command("ip", "rule", "add", "from", "all", "lookup", routeTable, "prio", rulePriority).Run()
}

func isInterfaceUp(iface string) bool {
	return exec.Command("ip", "link", "show", iface, "up").Run() == nil
}

func maskToCIDR(ip, mask string) (string, error) {
	ipAddr := net.ParseIP(ip)
	maskAddr := net.ParseIP(mask)
	if ipAddr == nil || maskAddr == nil {
		return "", fmt.Errorf("ip=%s mask=%s неверны", ip, mask)
	}
	ones, _ := net.IPv4Mask(maskAddr[12], maskAddr[13], maskAddr[14], maskAddr[15]).Size()
	return fmt.Sprintf("%s/%d", ipAddr.String(), ones), nil
}

func isValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

func removeDuplicates(list []string) []string {
	seen := make(map[string]struct{}, len(list))
	result := make([]string, 0, len(list))
	for _, item := range list {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// логгеры
func logWarn(format string, args ...any) { fmt.Fprintf(os.Stderr, "⚠ "+format+"\n", args...) }
func logInfo(format string, args ...any) { fmt.Fprintf(os.Stderr, "ℹ "+format+"\n", args...) }

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: vpn-router [update|start|stop]")
		os.Exit(1)
	}

	cfg, err := loadConfig("/opt/etc/vpn-router/config.yaml")
	if err != nil {
		logWarn("ошибка загрузки конфигурации: %v", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "update":
		routes := parseRoutes(cfg)
		if err := applyRoutes(cfg, routes); err != nil {
			logWarn("обновление маршрутов: %v", err)
			os.Exit(1)
		}
		fmt.Println("Маршруты обновлены и применены")
	case "start":
		data, _ := os.ReadFile(routesFilePath)
		if err := applyRoutes(cfg, strings.Split(strings.TrimSpace(string(data)), "\n")); err != nil {
			logWarn("применение маршрутов: %v", err)
			os.Exit(1)
		}
		fmt.Println("Маршруты применены")
	case "stop":
		if err := resetRouting(); err != nil {
			logWarn("удаление маршрутов: %v", err)
			os.Exit(1)
		}
		fmt.Println("Маршруты удалены")
	default:
		fmt.Println("Неизвестная команда. Использование: vpn-router [update|start|stop]")
		os.Exit(1)
	}
}

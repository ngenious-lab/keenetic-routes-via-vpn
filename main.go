package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Config представляет структуру конфигурационного файла
type Config struct {
	VPNInterface string   `yaml:"vpn_interface"`
	RepoDir      string   `yaml:"repo_dir"`
	Files        []string `yaml:"files"`
	IPs          []string `yaml:"ips"` // Новое поле для кастомных IP-сетей
}

// loadConfig загружает конфигурацию из YAML-файла
func loadConfig(path string) (Config, error) {
	var config Config
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("не удалось прочитать config.yaml: %v", err)
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("не удалось разобрать config.yaml: %v", err)
	}
	return config, nil
}

// updateRoutes обновляет маршруты из репозитория и кастомных IP
func updateRoutes(config Config) error {
	var routes []string

	// 1. Чтение маршрутов из .bat файлов
	for _, file := range config.Files {
		path := filepath.Join(config.RepoDir, file)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Предупреждение: не удалось прочитать %s: %v\n", path, err)
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "route ADD ") {
				parts := strings.Fields(line)
				if len(parts) >= 5 {
					ip := parts[2]
					mask := parts[4]
					cidr, err := maskToCIDR(ip, mask)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Предупреждение: неверный формат маршрута %s: %v\n", line, err)
						continue
					}
					routes = append(routes, cidr)
				}
			}
		}
	}

	// 2. Добавление кастомных IP-сетей из конфигурации
	for _, ip := range config.IPs {
		if isValidCIDR(ip) {
			routes = append(routes, ip)
		} else {
			fmt.Fprintf(os.Stderr, "Предупреждение: неверный формат CIDR %s, пропускаем\n", ip)
		}
	}

	// Удаление дубликатов
	routes = removeDuplicates(routes)

	// Сохранение маршрутов в current_routes.txt
	if err := ioutil.WriteFile("/opt/etc/vpn-router/current_routes.txt", []byte(strings.Join(routes, "\n")+"\n"), 0644); err != nil {
		return fmt.Errorf("не удалось записать current_routes.txt: %v", err)
	}
	return nil
}

// maskToCIDR преобразует IP и маску подсети в CIDR-нотацию
func maskToCIDR(ip, mask string) (string, error) {
	// Простая реализация, предполагает валидные входные данные
	// Реальная реализация должна парсить IP и маску и вычислять CIDR
	return ip + "/24", nil // Заглушка, замените на реальную логику
}

// isValidCIDR проверяет, является ли строка валидным CIDR
func isValidCIDR(cidr string) bool {
	// Заглушка, замените на реальную проверку (например, с использованием net.ParseCIDR)
	return strings.Contains(cidr, "/")
}

// removeDuplicates удаляет дубликаты из списка строк
func removeDuplicates(list []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range list {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// Пример функции main (упрощённая)
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: vpn-router [update|start|stop]")
		os.Exit(1)
	}

	config, err := loadConfig("/opt/etc/vpn-router/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки конфигурации: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "update":
		if err := updateRoutes(config); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка обновления маршрутов: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Маршруты успешно обновлены")
	case "start":
		// Логика применения маршрутов
	case "stop":
		// Логика удаления маршрутов
	default:
		fmt.Println("Неизвестная команда. Использование: vpn-router [update|start|stop]")
		os.Exit(1)
	}
}

package main

import (
	"fmt"
<<<<<<< HEAD
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

=======
>>>>>>> fc119a5d7c7abd52e470eee8a21efa84c77ebd82
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
<<<<<<< HEAD
	IPs          []string `yaml:"ips"`
=======
	IPs          []string `yaml:"ips"` // Новое поле для кастомных IP-сетей
>>>>>>> fc119a5d7c7abd52e470eee8a21efa84c77ebd82
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

<<<<<<< HEAD
// updateRoutes обновляет маршруты и применяет их в таблицу 1000
=======
// updateRoutes обновляет маршруты из репозитория и кастомных IP
>>>>>>> fc119a5d7c7abd52e470eee8a21efa84c77ebd82
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
<<<<<<< HEAD
	fmt.Fprintf(os.Stderr, "Маршруты успешно сохранены в /opt/etc/vpn-router/current_routes.txt\n")

	// 3. Проверка активности VPN-интерфейса
	if !isInterfaceUp(config.VPNInterface) {
		fmt.Fprintf(os.Stderr, "Предупреждение: VPN-интерфейс %s не активен. Пропускаем применение маршрутов.\n", config.VPNInterface)
		return nil
	}

	// 4. Применение маршрутов в таблицу 1000
	if err := startRoutes(config); err != nil {
		return fmt.Errorf("не удалось применить маршруты: %v", err)
	}

	return nil
}

// startRoutes применяет маршруты из current_routes.txt в таблицу 1000
func startRoutes(config Config) error {
	// Очищаем таблицу 1000
	cmd := exec.Command("ip", "route", "flush", "table", "1000")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Предупреждение: не удалось очистить таблицу маршрутов 1000: %v\n", err)
	}

	// Читаем маршруты из current_routes.txt
	data, err := ioutil.ReadFile("/opt/etc/vpn-router/current_routes.txt")
	if err != nil {
		return fmt.Errorf("не удалось прочитать current_routes.txt: %v", err)
	}
	lines := strings.Split(string(data), "\n")

	// Добавляем маршруты в таблицу 1000
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !isValidCIDR(line) {
			fmt.Fprintf(os.Stderr, "Предупреждение: неверный формат CIDR %s, пропускаем\n", line)
			continue
		}
		cmd := exec.Command("ip", "route", "add", line, "dev", config.VPNInterface, "table", "1000")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Предупреждение: не удалось добавить маршрут %s в таблицу 1000: %v\n", line, err)
		}
	}
	fmt.Fprintf(os.Stderr, "Маршруты успешно применены в таблицу 1000\n")
	return nil
}

// stopRoutes очищает таблицу маршрутов 1000
func stopRoutes(config Config) error {
	cmd := exec.Command("ip", "route", "flush", "table", "1000")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("не удалось очистить таблицу маршрутов 1000: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Таблица маршрутов 1000 успешно очищена\n")
	return nil
}

// isInterfaceUp проверяет, активен ли интерфейс
func isInterfaceUp(iface string) bool {
	cmd := exec.Command("ip", "link", "show", iface, "up")
	return cmd.Run() == nil
}

// maskToCIDR преобразует IP и маску подсети в CIDR-нотацию
func maskToCIDR(ip, mask string) (string, error) {
	// Заглушка, замените на реальную логику (например, с использованием net.IP)
	return ip + "/24", nil
=======
	return nil
}

// maskToCIDR преобразует IP и маску подсети в CIDR-нотацию
func maskToCIDR(ip, mask string) (string, error) {
	// Простая реализация, предполагает валидные входные данные
	// Реальная реализация должна парсить IP и маску и вычислять CIDR
	return ip + "/24", nil // Заглушка, замените на реальную логику
>>>>>>> fc119a5d7c7abd52e470eee8a21efa84c77ebd82
}

// isValidCIDR проверяет, является ли строка валидным CIDR
func isValidCIDR(cidr string) bool {
<<<<<<< HEAD
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
=======
	// Заглушка, замените на реальную проверку (например, с использованием net.ParseCIDR)
	return strings.Contains(cidr, "/")
>>>>>>> fc119a5d7c7abd52e470eee8a21efa84c77ebd82
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

<<<<<<< HEAD
// main обрабатывает команды update, start, stop
=======
// Пример функции main (упрощённая)
>>>>>>> fc119a5d7c7abd52e470eee8a21efa84c77ebd82
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
<<<<<<< HEAD
		fmt.Println("Маршруты успешно обновлены и применены")
	case "start":
		if err := startRoutes(config); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка применения маршрутов: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Маршруты успешно применены")
	case "stop":
		if err := stopRoutes(config); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка удаления маршрутов: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Маршруты успешно удалены")
=======
		fmt.Println("Маршруты успешно обновлены")
	case "start":
		// Логика применения маршрутов
	case "stop":
		// Логика удаления маршрутов
>>>>>>> fc119a5d7c7abd52e470eee8a21efa84c77ebd82
	default:
		fmt.Println("Неизвестная команда. Использование: vpn-router [update|start|stop]")
		os.Exit(1)
	}
}

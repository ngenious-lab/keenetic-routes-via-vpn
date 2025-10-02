package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config представляет структуру конфигурационного файла
type Config struct {
	VPNInterface string   `yaml:"vpn_interface"`
	RepoDir      string   `yaml:"repo_dir"`
	Files        []string `yaml:"files"`
	IPs          []string `yaml:"ips"`
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

// updateRoutes обновляет маршруты и применяет их в таблицу 1000
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
	// Проверка активности интерфейса
	if !isInterfaceUp(config.VPNInterface) {
		return fmt.Errorf("VPN-интерфейс %s не активен", config.VPNInterface)
	}

	// Очищаем таблицу 1000
	cmd := exec.Command("ip", "route", "flush", "table", "1000")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Предупреждение: не удалось очистить таблицу маршрутов 1000: %v\n", err)
	}

	// Проверяем, существует ли правило с приоритетом 1995
	cmd = exec.Command("ip", "rule", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Предупреждение: не удалось проверить правила маршрутизации: %v\n", err)
	} else if strings.Contains(string(output), "1995") {
		// Удаляем правило, если оно существует
		cmd = exec.Command("ip", "rule", "del", "priority", "1995", "2>/dev/null")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Предупреждение: не удалось удалить правило с приоритетом 1995: %v\n", err)
		}
	}

	// Добавляем правило маршрутизации
	cmd = exec.Command("ip", "rule", "add", "from", "all", "lookup", "1000", "prio", "1995")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("не удалось добавить правило маршрутизации (prio 1995): %v", err)
	}

	// Читаем маршруты из current_routes.txt
	data, err := ioutil.ReadFile("/opt/etc/vpn-router/current_routes.txt")
	if err != nil {
		return fmt.Errorf("не удалось прочитать current_routes.txt: %v", err)
	}
	lines := strings.Split(string(data), "\n")

	// Добавляем маршруты в таблицу 1000 с улучшенной обработкой ошибок
	successCount := 0
	failCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !isValidCIDR(line) {
			fmt.Fprintf(os.Stderr, "Предупреждение: неверный формат CIDR %s, пропускаем\n", line)
			failCount++
			continue
		}

		// Пробуем добавить маршрут как есть
		cmd := exec.Command("ip", "route", "add", "table", "1000", line, "dev", config.VPNInterface, "2>/dev/null")
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка: не удалось добавить маршрут %s: %v, output: %s\n", line, err, string(output))

			// Пробуем разбить /23 на два /24 если это необходимо
			if strings.Contains(string(output), "Invalid argument") && strings.HasSuffix(line, "/23") {
				fmt.Fprintf(os.Stderr, "Пробуем разбить %s на две подсети /24\n", line)

				// Разбиваем /23 на два /24
				ip, ipnet, err := net.ParseCIDR(line)
				if err == nil {
					// Первая подсеть /24
					ip1 := ip.Mask(net.CIDRMask(24, 32))
					cidr1 := fmt.Sprintf("%s/24", ip1.String())

					cmd1 := exec.Command("ip", "route", "add", "table", "1000", cidr1, "dev", config.VPNInterface, "2>/dev/null")
					if output1, err1 := cmd1.CombinedOutput(); err1 != nil {
						fmt.Fprintf(os.Stderr, "Ошибка добавления %s: %v, output: %s\n", cidr1, err1, string(output1))
					} else {
						fmt.Fprintf(os.Stderr, "Успешно добавлен маршрут: %s\n", cidr1)
						successCount++
					}

					// Вторая подсеть /24 (ip1 + 1)
					ip2 := make(net.IP, len(ip1))
					copy(ip2, ip1)
					ip2[3] += 1
					cidr2 := fmt.Sprintf("%s/24", ip2.String())

					cmd2 := exec.Command("ip", "route", "add", "table", "1000", cidr2, "dev", config.VPNInterface, "2>/dev/null")
					if output2, err2 := cmd2.CombinedOutput(); err2 != nil {
						fmt.Fprintf(os.Stderr, "Ошибка добавления %s: %v, output: %s\n", cidr2, err2, string(output2))
					} else {
						fmt.Fprintf(os.Stderr, "Успешно добавлен маршрут: %s\n", cidr2)
						successCount++
					}
				}
			} else {
				failCount++
			}
		} else {
			fmt.Fprintf(os.Stderr, "Успешно добавлен маршрут: %s\n", line)
			successCount++
		}
	}

	fmt.Fprintf(os.Stderr, "Маршруты применены: %d успешно, %d с ошибками\n", successCount, failCount)
	return nil
}

// stopRoutes очищает таблицу маршрутов 1000
func stopRoutes(config Config) error {
	// Очищаем таблицу 1000
	cmd := exec.Command("ip", "route", "flush", "table", "1000")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Предупреждение: не удалось очистить таблицу маршрутов 1000: %v\n", err)
	}

	// Проверяем, существует ли правило с приоритетом 1995
	cmd = exec.Command("ip", "rule", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Предупреждение: не удалось проверить правила маршрутизации: %v\n", err)
	} else if strings.Contains(string(output), "1995") {
		// Удаляем правило, если оно существует
		cmd = exec.Command("ip", "rule", "del", "priority", "1995")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Предупреждение: не удалось удалить правило с приоритетом 1995: %v\n", err)
		}
	}

	fmt.Fprintf(os.Stderr, "Таблица маршрутов 1000 и правила успешно очищены\n")
	return nil
}

// isInterfaceUp проверяет, активен ли интерфейс
func isInterfaceUp(iface string) bool {
	cmd := exec.Command("ip", "link", "show", iface, "up")
	return cmd.Run() == nil
}

// maskToCIDR преобразует IP и маску подсети в CIDR-нотацию
func maskToCIDR(ip, mask string) (string, error) {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return "", fmt.Errorf("неверный IP: %s", ip)
	}
	maskAddr := net.ParseIP(mask)
	if maskAddr == nil {
		return "", fmt.Errorf("неверная маска: %s", mask)
	}
	ones, _ := net.IPv4Mask(maskAddr[12], maskAddr[13], maskAddr[14], maskAddr[15]).Size()
	return fmt.Sprintf("%s/%d", ip, ones), nil
}

// isValidCIDR проверяет, является ли строка валидным CIDR
func isValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
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

// main обрабатывает команды update, start, stop
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
	default:
		fmt.Println("Неизвестная команда. Использование: vpn-router [update|start|stop]")
		os.Exit(1)
	}
}

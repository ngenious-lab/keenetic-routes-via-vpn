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
	configPath      = "/opt/etc/vpn-router/config.yaml"
	routesFilePath  = "/opt/etc/vpn-router/current_routes.txt"
	routeTable      = "1000"
	rulePriority    = "1995"
	defaultRepoDir  = "/opt/etc/ip-address"
)

type Config struct {
	VPNInterface string   `yaml:"vpn_interface"`
	RepoDir      string   `yaml:"repo_dir"`
	Files        []string `yaml:"files"`
	IPs          []string `yaml:"ips"`
}

func loadConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	// укажем дефолтный repo_dir если не задан
	if cfg.RepoDir == "" {
		cfg.RepoDir = defaultRepoDir
	}
	return cfg, nil
}

func writeLinesToFile(path string, lines []string) error {
	if len(lines) == 0 {
		// всегда пишем пустой файл с newline, чтобы downstream не ломался
		return os.WriteFile(path, []byte(""), 0644)
	}
	data := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(data), 0644)
}

func runCommandCaptureOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func isInterfaceUp(iface string) bool {
	// ip link show <iface> up -> exit 0 если up
	cmd := exec.Command("ip", "link", "show", iface, "up")
	return cmd.Run() == nil
}

func maskToCIDR(ip, mask string) (string, error) {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return "", fmt.Errorf("invalid ip: %s", ip)
	}
	maskAddr := net.ParseIP(mask)
	if maskAddr == nil {
		return "", fmt.Errorf("invalid mask: %s", mask)
	}
	// для IPv4: берем последние 4 байта
	mask4 := net.IPv4Mask(maskAddr[12], maskAddr[13], maskAddr[14], maskAddr[15])
	ones, _ := mask4.Size()
	return fmt.Sprintf("%s/%d", ipAddr.String(), ones), nil
}

func isValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

func removeDuplicates(list []string) []string {
	seen := make(map[string]struct{}, len(list))
	var out []string
	for _, v := range list {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// parseRoutes читает .bat файлы и конфиг. Возвращает уникальный список CIDR.
func parseRoutes(cfg Config) []string {
	var routes []string
	// парсинг файлов в repo_dir
	for _, f := range cfg.Files {
		path := filepath.Join(cfg.RepoDir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Предупреждение: не удалось прочитать %s: %v\n", path, err)
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, ln := range lines {
			ln = strings.TrimSpace(ln)
			if strings.HasPrefix(ln, "route ADD ") || strings.HasPrefix(ln, "route add ") {
				parts := strings.Fields(ln)
				// ожидаем: route ADD <ip> MASK <mask> gateway ...
				// в исходном коде использовался индекс parts[2] и parts[4]
				if len(parts) >= 5 {
					ip := parts[2]
					mask := parts[4]
					if cidr, err := maskToCIDR(ip, mask); err == nil {
						routes = append(routes, cidr)
					} else {
						fmt.Fprintf(os.Stderr, "Предупреждение: неверный маршрут в %s: %v\n", ln, err)
					}
				}
			}
		}
	}

	// кастомные IP из конфигурации (CIDR)
	for _, ip := range cfg.IPs {
		if isValidCIDR(ip) {
			routes = append(routes, ip)
		} else {
			fmt.Fprintf(os.Stderr, "Предупреждение: неверный формат CIDR %s, пропускаем\n", ip)
		}
	}

	return removeDuplicates(routes)
}

// applyRoutes сохраняет current_routes.txt и применяет маршруты (если интерфейс активен).
func applyRoutes(cfg Config, routes []string) error {
	// записать файл
	if err := writeLinesToFile(routesFilePath, routes); err != nil {
		return fmt.Errorf("запись %s: %w", routesFilePath, err)
	}
	fmt.Fprintln(os.Stderr, "Маршруты сохранены в", routesFilePath)

	// если интерфейс не up — просто выйдем (не применяем маршруты)
	if cfg.VPNInterface == "" {
		return fmt.Errorf("vpn_interface не указан в конфиге")
	}
	if !isInterfaceUp(cfg.VPNInterface) {
		fmt.Fprintf(os.Stderr, "VPN-интерфейс %s не активен — пропускаем применение маршрутов\n", cfg.VPNInterface)
		return nil
	}

	// применим: очистим таблицу, поправим правило, добавим записи
	if err := exec.Command("ip", "route", "flush", "table", routeTable).Run(); err != nil {
		// не фатальная ошибка — логируем
		fmt.Fprintf(os.Stderr, "Предупреждение: не удалось очистить таблицу %s: %v\n", routeTable, err)
	}

	// удалим правило 1995 если есть
	if out, err := runCommandCaptureOutput("ip", "rule", "show"); err == nil {
		if strings.Contains(out, rulePriority) {
			_ = exec.Command("ip", "rule", "del", "priority", rulePriority).Run()
		}
	}

	// добавим правило на lookup tableTable с prio
	if err := exec.Command("ip", "rule", "add", "from", "all", "lookup", routeTable, "prio", rulePriority).Run(); err != nil {
		return fmt.Errorf("не удалось добавить ip rule prio %s: %w", rulePriority, err)
	}

	// прочитаем файл для добавления маршрутов (на случай если routes приходили пустыми)
	data, err := os.ReadFile(routesFilePath)
	if err != nil {
		return fmt.Errorf("чтение %s: %w", routesFilePath, err)
	}
	lines := strings.Split(string(data), "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if !isValidCIDR(ln) {
			fmt.Fprintf(os.Stderr, "Предупреждение: неверный CIDR %s, пропускаем\n", ln)
			continue
		}
		// ip route add table 1000 <cidr> dev <vpn_iface>
		if err := exec.Command("ip", "route", "add", "table", routeTable, ln, "dev", cfg.VPNInterface).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Предупреждение: не удалось добавить маршрут %s: %v\n", ln, err)
			// продолжаем дальше
		}
	}
	fmt.Fprintln(os.Stderr, "Маршруты применены в таблицу", routeTable)
	return nil
}

// startRoutes применяет маршруты из файла current_routes.txt (без повторного парсинга)
func startRoutes(cfg Config) error {
	if cfg.VPNInterface == "" {
		return fmt.Errorf("vpn_interface не указан в конфиге")
	}
	if !isInterfaceUp(cfg.VPNInterface) {
		return fmt.Errorf("VPN интерфейс %s не активен", cfg.VPNInterface)
	}

	// очистим таблицу
	_ = exec.Command("ip", "route", "flush", "table", routeTable).Run()

	// удалим правило если есть
	if out, err := runCommandCaptureOutput("ip", "rule", "show"); err == nil {
		if strings.Contains(out, rulePriority) {
			_ = exec.Command("ip", "rule", "del", "priority", rulePriority).Run()
		}
	}

	// добавим правило
	if err := exec.Command("ip", "rule", "add", "from", "all", "lookup", routeTable, "prio", rulePriority).Run(); err != nil {
		return fmt.Errorf("не удалось добавить ip rule: %w", err)
	}

	// прочитаем файл и добавим маршруты
	data, err := os.ReadFile(routesFilePath)
	if err != nil {
		return fmt.Errorf("чтение %s: %w", routesFilePath, err)
	}
	lines := strings.Split(string(data), "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if !isValidCIDR(ln) {
			fmt.Fprintf(os.Stderr, "Пропускаем неверный CIDR %s\n", ln)
			continue
		}
		_ = exec.Command("ip", "route", "add", "table", routeTable, ln, "dev", cfg.VPNInterface).Run()
	}
	fmt.Fprintln(os.Stderr, "Маршруты применены (start).")
	return nil
}

func stopRoutes(cfg Config) error {
	_ = exec.Command("ip", "route", "flush", "table", routeTable).Run()
	// удаляем правило, если есть
	_ = exec.Command("ip", "rule", "del", "priority", rulePriority).Run()
	fmt.Fprintln(os.Stderr, "Таблица маршрутов", routeTable, "и правило очищены.")
	return nil
}

// updateCommand: парсим repository файлы, сохраняем и применяем
func updateCommand(cfg Config) error {
	routes := parseRoutes(cfg)
	return applyRoutes(cfg, routes)
}

// updateRepoCommand: делает git pull в repo_dir и затем вызывает update
func updateRepoCommand(cfg Config) error {
	repo := cfg.RepoDir
	if repo == "" {
		repo = defaultRepoDir
	}
	// проверяем каталог
	if _, err := os.Stat(repo); os.IsNotExist(err) {
		return fmt.Errorf("репозиторий %s не найден", repo)
	}
	out, err := runCommandCaptureOutput("git", "-C", repo, "pull")
	if err != nil {
		return fmt.Errorf("git pull error: %v - %s", err, out)
	}
	fmt.Fprintln(os.Stderr, "git pull:", out)
	// затем обновляем маршруты
	return updateCommand(cfg)
}

func statusCommand(cfg Config) error {
	// выводим правило и таблицу
	if out, err := runCommandCaptureOutput("ip", "rule", "show"); err == nil {
		fmt.Println("ip rule show:")
		fmt.Println(out)
	} else {
		fmt.Println("ip rule show: error:", err)
	}
	if out, err := runCommandCaptureOutput("ip", "route", "show", "table", routeTable); err == nil {
		fmt.Println("\nip route show table", routeTable, ":")
		fmt.Println(out)
	} else {
		fmt.Println("\nip route show table", routeTable, "error:", err)
	}
	return nil
}

func usage() {
	fmt.Println("Использование: vpn-router [update|start|stop|status|restart|update-repo]")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		// если конфиг не найден — сообщим, но некоторые команды (update-repo) всё ещё могут работать, т.к. нужен только repo_dir
		fmt.Fprintf(os.Stderr, "Внимание: не удалось загрузить конфиг %s: %v\n", configPath, err)
		// создаём минимальный cfg для update-repo/status, если нужно
		cfg = Config{RepoDir: defaultRepoDir}
	}

	cmd := os.Args[1]
	switch cmd {
	case "update":
		if err := updateCommand(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка update: %v\n", err)
			os.Exit(2)
		}
		fmt.Println("Маршруты обновлены и применены.")
	case "start":
		if err := startRoutes(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка start: %v\n", err)
			os.Exit(3)
		}
		fmt.Println("Маршруты применены.")
	case "stop":
		if err := stopRoutes(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка stop: %v\n", err)
			os.Exit(4)
		}
		fmt.Println("Маршруты удалены.")
	case "status":
		if err := statusCommand(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка status: %v\n", err)
			os.Exit(5)
		}
	case "restart":
		if err := stopRoutes(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка при stop в restart: %v\n", err)
		}
		if err := startRoutes(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка при start в restart: %v\n", err)
			os.Exit(6)
		}
		fmt.Println("Restart выполнен.")
	case "update-repo":
		if err := updateRepoCommand(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка update-repo: %v\n", err)
			os.Exit(7)
		}
		fmt.Println("Репозиторий обновлён и маршруты применены.")
	default:
		usage()
		os.Exit(1)
	}
}

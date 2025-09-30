package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	VPNInterface string   `yaml:"vpn_interface"`
	RepoDir      string   `yaml:"repo_dir"`
	Files        []string `yaml:"files"`
}

var config Config
var configFile = "/opt/etc/vpn-router/config.yaml"
var routesFile = "/opt/etc/vpn-router/current_routes.txt"

func main() {
	log.SetOutput(os.Stdout)

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal("Can't read config: ", err)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal("Bad config: ", err)
	}

	if len(os.Args) < 2 {
		log.Fatal("Need command: update, start, stop")
	}

	cmd := os.Args[1]
	switch cmd {
	case "update":
		updateRoutes()
	case "start":
		addRoutes()
	case "stop":
		deleteRoutes()
	default:
		log.Fatal("Unknown command: ", cmd)
	}
}

func updateRoutes() {
	cmd := exec.Command("git", "-C", config.RepoDir, "pull")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Git pull failed: %s %s - keeping old routes", err, output)
		return
	}

	var newRoutes []string
	for _, f := range config.Files {
		path := filepath.Join(config.RepoDir, f)
		file, err := os.Open(path)
		if err != nil {
			log.Printf("Warning: File %s not found: %s - skipping", f, err)
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			ip, mask := parseRouteLine(line)
			if ip == "" {
				continue
			}
			prefix := maskToPrefix(mask)
			if prefix == -1 {
				continue
			}
			newRoutes = append(newRoutes, ip+"/"+strconv.Itoa(prefix))
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Warning: Error scanning %s: %s - skipping partial", f, err)
		}
	}

	if len(newRoutes) == 0 {
		log.Println("No new routes parsed - keeping old")
		return
	}

	err = ioutil.WriteFile(routesFile, []byte(strings.Join(newRoutes, "\n")+"\n"), 0644)
	if err != nil {
		log.Printf("Failed to save new routes: %s - keeping old", err)
		return
	}
	log.Println("Routes updated successfully")
}

var routeRe = regexp.MustCompile(`(?i)route\s+ADD\s+(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\s+MASK\s+(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)

func parseRouteLine(line string) (ip, mask string) {
	m := routeRe.FindStringSubmatch(line)
	if len(m) == 3 {
		return m[1], m[2]
	}
	return "", ""
}

func maskToPrefix(mask string) int {
	parts := strings.Split(mask, ".")
	if len(parts) != 4 {
		return -1
	}
	var bits int
	for _, p := range parts {
		o, err := strconv.Atoi(p)
		if err != nil {
			return -1
		}
		for o != 0 {
			bits++
			o &= (o - 1)
		}
	}
	return bits
}

func addRoutes() {
	data, err := ioutil.ReadFile(routesFile)
	if err != nil {
		log.Printf("No routes file: %s - nothing to add", err)
		return
	}
	routes := strings.Split(string(data), "\n")

	execCmd("ip", "route", "flush", "table", "1000")
	execCmd("ip", "route", "add", "local", "default", "dev", "lo", "table", "1000")
	for _, r := range routes {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		execCmd("ip", "route", "add", r, "dev", config.VPNInterface, "table", "1000")
	}
	execCmd("ip", "rule", "del", "priority", "1995")
	execCmd("ip", "rule", "add", "table", "1000", "priority", "1995")
	log.Println("Routes added")
}

func deleteRoutes() {
	execCmd("ip", "route", "flush", "table", "1000")
	execCmd("ip", "rule", "del", "priority", "1995")
	log.Println("Routes deleted")
}

func execCmd(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Command '%s %s' failed: %s %s", name, strings.Join(arg, " "), err, output)
	}
}
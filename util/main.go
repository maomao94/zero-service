package main

import (
	"bufio"
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// Service represents the structure of a service
type Service struct {
	Name   string `yaml:"name"`
	Remark string `yaml:"remark,omitempty"` // Optional remark
}

// ServerConfig represents the structure of server configuration
type ServerConfig struct {
	SSHUser     string    `yaml:"sshUser"`
	SSHHost     string    `yaml:"sshHost"`
	SSHPort     string    `yaml:"sshPort"`
	SSHPassword string    `yaml:"sshPassword"`
	Path        string    `yaml:"path"`
	Services    []Service `yaml:"serviceName"`
	Remark      string    `yaml:"remark"`
}

// Config represents the overall configuration structure
type Config struct {
	Servers map[string]ServerConfig `yaml:"servers"`
}

func main() {
	// Display author information
	fmt.Println("====================================")
	fmt.Println("Welcome to the Service Management Tool")
	fmt.Println("Author: He Hanpeng")
	fmt.Println("Email: hehanpengyy@163.com")
	fmt.Println("====================================")

	// Define command line flags
	configFile := flag.String("f", "config.yaml", "Path to the YAML configuration file")
	flag.Parse()

	// Read the configuration from the specified file
	config := readConfig(*configFile)

	fmt.Println("====================================")
	fmt.Println("Available operations:")
	fmt.Println("1) run")
	fmt.Println("2) check")
	fmt.Println("3) exec")
	fmt.Println("4) log")
	fmt.Println("5) save")
	fmt.Print("Select an operation: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	operation := scanner.Text()

	fmt.Println("====================================")
	fmt.Println("Available servers:")
	serverNames := make([]string, 0, len(config.Servers))
	num := 1
	for i, name := range config.Servers {
		serverNames = append(serverNames, i)
		fmt.Printf("%d) %s\n", num, i+" "+name.Remark) // 显示序号和服务器名称
		num++
	}

	fmt.Print("Select a server by number: ")
	scanner.Scan()
	selectedServerIndex := scanner.Text()

	// 转换输入的序号为索引
	index, err := strconv.Atoi(selectedServerIndex)
	if err != nil || index < 1 || index > len(config.Servers) {
		fmt.Println("Invalid selection.")
		return
	}

	// 获取对应的服务器名称
	selectedServer := serverNames[index-1] // 序号从1开始，因此需要减去1

	serverConfig, exists := config.Servers[selectedServer]
	if !exists {
		fmt.Println("Server not found.")
		return
	}

	fmt.Printf("Selected server: %s (%s:%s)\n", selectedServer, serverConfig.SSHHost, serverConfig.SSHPort)

	switch operation {
	case "1":
		runServices(serverConfig)
	case "2":
		checkServices(serverConfig)
	case "3":
		execService(serverConfig)
	case "4":
		logService(serverConfig)
	case "5":
		saveImages(serverConfig)
	default:
		fmt.Println("Invalid operation.")
	}
}

// readConfig reads the YAML configuration file
func readConfig(filename string) Config {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		panic(err)
	}
	return config
}

// runServices runs the specified services
func runServices(serverConfig ServerConfig) {
	fmt.Println("====================================")
	fmt.Println("Select the mode:")
	fmt.Println("1) single (single selection)")
	fmt.Println("2) multi (multiple selection)")
	fmt.Print("Select mode: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	mode := scanner.Text()

	fmt.Println("Available services:")
	for i, service := range serverConfig.Services {
		fmt.Printf("%d) %s (%s)\n", i+1, service.Name, service.Remark)
	}

	var selectedServices []string
	if mode == "1" { // Single selection
		fmt.Print("Select a service to run: ")
		scanner.Scan()
		serviceIndex := scanner.Text()
		if i := parseIndex(serviceIndex, len(serverConfig.Services)); i != -1 {
			selectedServices = append(selectedServices, serverConfig.Services[i].Name)
		} else {
			fmt.Println("Invalid service index.")
			return
		}
	} else if mode == "2" { // Multiple selection
		fmt.Print("Select service(s) to run (separated by space): ")
		scanner.Scan()
		serviceIndexes := strings.Fields(scanner.Text())

		for _, index := range serviceIndexes {
			if i := parseIndex(index, len(serverConfig.Services)); i != -1 {
				selectedServices = append(selectedServices, serverConfig.Services[i].Name)
			} else {
				fmt.Printf("Invalid service index: %s\n", index)
			}
		}
	}

	if len(selectedServices) == 0 {
		fmt.Println("No valid services selected.")
		return
	}

	fmt.Println("====================================")
	fmt.Println("Select the action:")
	fmt.Println("1) start")
	fmt.Println("2) stop")
	fmt.Println("3) up")
	fmt.Println("4) restart")
	fmt.Print("Select action: ")
	scanner2 := bufio.NewScanner(os.Stdin)
	scanner2.Scan()
	actionNum := scanner2.Text()
	var action = ""
	switch actionNum {
	case "1":
		action = "start"
	case "2":
		action = "stop"
	case "3":
		action = "up -d"
	case "4":
		action = "restart"
	}

	// Print the command to be executed11
	command := fmt.Sprintf("sshpass -p '%s' ssh -p %s %s@%s 'docker compose -f %s %s %s'",
		serverConfig.SSHPassword, serverConfig.SSHPort, serverConfig.SSHUser, serverConfig.SSHHost, serverConfig.Path, action, strings.Join(selectedServices, " "))

	//command := fmt.Sprintf("docker compose -f %s up -d %s", serverConfig.Path, strings.Join(selectedServices, " "))
	fmt.Println("Executing command:", command)

	// Confirm execution
	if confirmExecution() {
		startTime := time.Now() // Start time
		output := executeCommand(command)
		fmt.Println("====================================")
		fmt.Println(output)
		elapsedTime := time.Since(startTime) // Calculate elapsed time
		fmt.Printf("Command executed in: %s\n", formatDuration(elapsedTime))
	} else {
		fmt.Println("Command execution cancelled.")
	}
}

// checkServices checks the status of the services
func checkServices(serverConfig ServerConfig) {
	fmt.Println("====================================")
	fmt.Println("Available services:")
	for i, service := range serverConfig.Services {
		fmt.Printf("%d) %s (%s)\n", i+1, service.Name, service.Remark)
	}

	fmt.Print("Select service(s) to check (separated by space): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	serviceIndexes := strings.Fields(scanner.Text())

	selectedServices := make([]string, 0)
	for _, index := range serviceIndexes {
		if i := parseIndex(index, len(serverConfig.Services)); i != -1 {
			selectedServices = append(selectedServices, serverConfig.Services[i].Name)
		} else {
			fmt.Printf("Invalid service index: %s\n", index)
		}
	}

	if len(selectedServices) == 0 {
		fmt.Println("No valid services selected.")
		return
	}

	// Print the command to be executed
	command := fmt.Sprintf("sshpass -p '%s' ssh -p %s %s@%s 'cd %s && docker-compose ps'",
		serverConfig.SSHPassword, serverConfig.SSHPort, serverConfig.SSHUser, serverConfig.SSHHost, serverConfig.Path)
	fmt.Println("Executing command:", command)

	// Confirm execution
	if confirmExecution() {
		output := executeCommand(command)
		fmt.Println("====================================")
		fmt.Println("Service Status:")
		fmt.Println(output)
	} else {
		fmt.Println("Command execution cancelled.")
	}
}

// execService enters the specified service container
func execService(serverConfig ServerConfig) {
	fmt.Println("====================================")
	fmt.Println("Available services:")
	for i, service := range serverConfig.Services {
		fmt.Printf("%d) %s (%s)\n", i+1, service.Name, service.Remark)
	}

	fmt.Print("Select a service to exec: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	serviceIndex := scanner.Text()

	if i := parseIndex(serviceIndex, len(serverConfig.Services)); i != -1 {
		service := serverConfig.Services[i]
		// Print the command to be executed
		command := fmt.Sprintf("sshpass -p '%s' ssh -p %s %s@%s 'cd %s && docker-compose exec %s sh'",
			serverConfig.SSHPassword, serverConfig.SSHPort, serverConfig.SSHUser, serverConfig.SSHHost, serverConfig.Path, service.Name)
		fmt.Println("Executing command:", command)
		executeInteractiveCommand(command)
	} else {
		fmt.Println("Invalid service index.")
	}
}

// logService views the logs of the specified service
func logService(serverConfig ServerConfig) {
	fmt.Println("====================================")
	fmt.Println("Available services:")
	for i, service := range serverConfig.Services {
		fmt.Printf("%d) %s (%s)\n", i+1, service.Name, service.Remark)
	}

	fmt.Print("Select a service to view logs: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	serviceIndex := scanner.Text()

	if i := parseIndex(serviceIndex, len(serverConfig.Services)); i != -1 {
		service := serverConfig.Services[i]
		command := fmt.Sprintf("sshpass -p '%s' ssh -p %s %s@%s 'cd %s && docker-compose logs %s'",
			serverConfig.SSHPassword, serverConfig.SSHPort, serverConfig.SSHUser, serverConfig.SSHHost, serverConfig.Path, service.Name)
		fmt.Println("Executing command:", command)
		output := executeCommand(command)
		fmt.Println("====================================")
		fmt.Println("Service Logs:")
		fmt.Println(output)
	} else {
		fmt.Println("Invalid service index.")
	}
}

// saveImages saves the images of the specified services
func saveImages(serverConfig ServerConfig) {
	fmt.Println("====================================")
	fmt.Println("Available services:")
	for i, service := range serverConfig.Services {
		fmt.Printf("%d) %s (%s)\n", i+1, service.Name, service.Remark)
	}

	fmt.Print("Select service(s) to save (separated by space): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	serviceIndexes := strings.Fields(scanner.Text())

	selectedServices := make([]string, 0)
	for _, index := range serviceIndexes {
		if i := parseIndex(index, len(serverConfig.Services)); i != -1 {
			selectedServices = append(selectedServices, serverConfig.Services[i].Name)
		} else {
			fmt.Printf("Invalid service index: %s\n", index)
		}
	}

	if len(selectedServices) == 0 {
		fmt.Println("No valid services selected.")
		return
	}

	// Print the command to be executed
	command := fmt.Sprintf("sshpass -p '%s' ssh -p %s %s@%s 'cd %s && docker save -o images.tar %s'",
		serverConfig.SSHPassword, serverConfig.SSHPort, serverConfig.SSHUser, serverConfig.SSHHost, serverConfig.Path, strings.Join(selectedServices, " "))
	fmt.Println("Executing command:", command)

	// Confirm execution
	if confirmExecution() {
		startTime := time.Now() // Start time
		executeCommand(command)
		elapsedTime := time.Since(startTime) // Calculate elapsed time
		fmt.Printf("Command executed in: %s\n", formatDuration(elapsedTime))
	} else {
		fmt.Println("Command execution cancelled.")
	}
}

// parseIndex parses the string index to integer
func parseIndex(indexStr string, length int) int {
	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 1 || index > length {
		return -1
	}
	return index - 1 // Adjust for zero-based index
}

// confirmExecution prompts the user for confirmation
func confirmExecution() bool {
	fmt.Print("Are you sure you want to execute this command? (y/n): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.ToLower(scanner.Text()) == "y"
}

// formatDuration formats the duration to a string
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// executeCommand executes a shell command and returns the output
func executeCommand(command string) string {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing command: %s\n", err)
	}
	return string(output)
}

// executeInteractiveCommand executes an interactive shell command
func executeInteractiveCommand(command string) {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing command: %s\n", err)
	}
}

// Execute remote command via SSH
func executeRemoteCommand(config ServerConfig, command string) string {
	// Create the SSH client configuration
	sshConfig := &ssh.ClientConfig{
		User: config.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.SSHPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Insecure, for testing only
	}

	// Build SSH connection string
	sshAddress := fmt.Sprintf("%s:%s", config.SSHHost, config.SSHPort)

	// Establish SSH connection
	client, err := ssh.Dial("tcp", sshAddress, sshConfig)
	if err != nil {
		fmt.Println("Failed to dial: ", err)
		return ""
	}
	defer client.Close()

	// Create a new session
	session, err := client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return ""
	}
	defer session.Close()

	// Run the command on the remote server
	output, err := session.CombinedOutput(command)
	if err != nil {
		fmt.Printf("Failed to execute command: %s\n", err)
		return ""
	}

	return string(output)
}

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/creack/pty"
)

// 容器信息结构体
type ContainerInfo struct {
	ID      string
	Image   string
	Command string
	Created string
	Status  string
	Ports   string
	Name    string
}

func main() {
	options := []string{"start", "stop", "restart", "exec", "log"}
	fmt.Println("选择一个操作:")
	for i, option := range options {
		fmt.Printf("%d. %s\n", i+1, option)
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("请输入操作序号: ")
	scanner.Scan()
	actionIndex, _ := strconv.Atoi(scanner.Text())
	if actionIndex < 1 || actionIndex > len(options) {
		fmt.Println("无效选择")
		return
	}

	action := options[actionIndex-1]

	containers := getAllContainers()
	if len(containers) == 0 {
		fmt.Println("没有找到容器")
		return
	}

	fmt.Printf("%-5s %-15s %-30s %-30s %-20s %-25s %-20s %s\n", "序号", "CONTAINER ID", "IMAGE", "COMMAND", "CREATED", "STATUS", "PORTS", "NAMES")
	for i, container := range containers {
		fmt.Printf("%-5d %-15s %-30s %-30s %-20s %-25s %-20s %s\n",
			i+1, container.ID, container.Image, container.Command, container.Created, container.Status, container.Ports, container.Name)
	}

	fmt.Print("请输入容器序号: ")
	scanner.Scan()
	containerIndex, _ := strconv.Atoi(scanner.Text())
	if containerIndex < 1 || containerIndex > len(containers) {
		fmt.Println("无效选择")
		return
	}

	container := containers[containerIndex-1]

	// 使用伪终端执行命令
	executeCommandWithPTY(action, container.Name)
}

// 获取所有容器的详细信息
func getAllContainers() []ContainerInfo {
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{.ID}}|{{.Image}}|{{.Command}}|{{.CreatedAt}}|{{.Status}}|{{.Ports}}|{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("获取容器列表失败:", err)
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var containers []ContainerInfo
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) < 7 {
			continue
		}
		container := ContainerInfo{
			ID:      parts[0],
			Image:   parts[1],
			Command: strings.Trim(parts[2], `"`),
			Created: parts[3],
			Status:  parts[4],
			Ports:   parts[5],
			Name:    parts[6],
		}
		containers = append(containers, container)
	}
	return containers
}

// 使用伪终端执行命令
func executeCommandWithPTY(action, container string) {
	var cmd *exec.Cmd
	switch action {
	case "start":
		cmd = exec.Command("docker", "start", container)
	case "stop":
		cmd = exec.Command("docker", "stop", container)
	case "restart":
		cmd = exec.Command("docker", "restart", container)
	case "exec":
		// 创建伪终端用于执行 `exec -it`
		cmd = exec.Command("docker", "exec", "-i", container, "/bin/bash")
		ptmx, err := pty.Start(cmd)
		if err != nil {
			fmt.Printf("创建伪终端失败: %v\n", err)
			return
		}
		defer func() { _ = ptmx.Close() }()

		// 将标准输入输出与伪终端关联
		go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
		_, _ = io.Copy(os.Stdout, ptmx)
		return
	case "log":
		cmd = exec.Command("docker", "logs", "--tail", "100", "-f", container)
	default:
		fmt.Println("未知操作")
		return
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("执行 %s 命令失败: %v\n", action, err)
	} else {
		fmt.Printf("%s 操作成功执行\n", action)
	}
}

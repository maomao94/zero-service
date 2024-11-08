package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
)

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
	fmt.Println("Welcome to the Service Management Tool")
	fmt.Println("Author: He Hanpeng")
	fmt.Println("Email: hehanpengyy@163.com")
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

	// 设置 tabwriter 来美化输出格式
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "序号\tCONTAINER ID\tIMAGE\tCOMMAND\tCREATED\tSTATUS\tPORTS\tNAMES\n")
	for i, container := range containers {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			i+1, container.ID, container.Image, container.Command, container.Created, container.Status, container.Ports, container.Name)
	}
	w.Flush() // 刷新缓冲区，将内容打印到控制台

	fmt.Print("请输入容器序号: ")
	scanner.Scan()
	containerIndex, _ := strconv.Atoi(scanner.Text())
	if containerIndex < 1 || containerIndex > len(containers) {
		fmt.Println("无效选择")
		return
	}

	container := containers[containerIndex-1]
	executeCommandWithInteractiveTTY(action, container.Name)
}

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

func executeCommandWithInteractiveTTY(action, container string) {
	var cmd *exec.Cmd
	switch action {
	case "start":
		cmd = exec.Command("docker", "start", container)
	case "stop":
		cmd = exec.Command("docker", "stop", container)
	case "restart":
		cmd = exec.Command("docker", "restart", container)
	case "exec":
		// 使用 `-it` 伪终端参数
		cmd = exec.Command("docker", "exec", "-it", container, "/bin/bash")
	case "log":
		cmd = exec.Command("docker", "logs", "--tail", "100", "-f", container)
	default:
		fmt.Println("未知操作")
		return
	}

	// 将标准输入、输出和错误与当前终端关联
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("执行 %s 命令失败: %v\n", action, err)
	} else {
		fmt.Printf("%s 操作成功执行\n", action)
	}
}

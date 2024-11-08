package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
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
	options := []string{"start", "stop", "restart", "exec", "log", "images"}
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

	// 获取用户输入的容器名称过滤条件（可选）
	fmt.Print("请输入容器名称过滤条件（可选，留空表示不过滤）: ")
	scanner.Scan()
	filter := scanner.Text()

	containers := getAllContainers(filter) // 传入过滤条件
	if len(containers) == 0 {
		fmt.Println("没有找到容器")
		return
	}

	// 设置 tabwriter 来美化输出格式
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "序号\tCONTAINER ID\tIMAGE\tCOMMAND\tCREATED\tSTATUS\tPORTS\tNAMES\n")
	for i, container := range containers {
		// 格式化时间为简洁的日期格式
		createdTime, err := time.Parse("2006-01-02T15:04:05Z07:00", container.Created)
		if err != nil {
			createdTime = time.Now() // 如果解析错误，使用当前时间
		}
		// 格式化为日期和时间（YYYY-MM-DD HH:MM:SS）
		formattedCreated := createdTime.Format("2006-01-02 15:04:05")

		// 输出容器信息
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			i+1, container.ID, container.Image, container.Command, formattedCreated, container.Status, container.Ports, container.Name)
	}
	w.Flush() // 刷新缓冲区，将内容打印到控制台

	// 容器操作部分
	if action != "images" {
		fmt.Print("请输入容器序号: ")
		scanner.Scan()
		containerIndex, _ := strconv.Atoi(scanner.Text())
		if containerIndex < 1 || containerIndex > len(containers) {
			fmt.Println("无效选择")
			return
		}

		container := containers[containerIndex-1]
		executeCommandWithInteractive(action, container.Name)
	}
}

func getAllContainers(filter string) []ContainerInfo {
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

		// 如果 filter 不为空，检查容器的 Name 或 Image 是否包含 filter 字符串
		if filter != "" && !strings.Contains(parts[6], filter) && !strings.Contains(parts[1], filter) {
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

func executeCommandWithInteractive(action, container string) {
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

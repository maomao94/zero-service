package main

import (
	"bufio"
	"fmt"
	"golang.org/x/term"
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
	// 欢迎信息、作者信息、分隔线部分
	fmt.Println("\033[1;32mWelcome to the Service Management Tool v1.0.0\033[0m") // 绿色
	fmt.Println("\033[1;32mAuthor: He Hanpeng\033[0m")                            // 绿色
	fmt.Println("\033[1;32mEmail: hehanpengyy@163.com\033[0m")                    // 绿色
	fmt.Println(strings.Repeat("=", getTerminalWidth()))                          // 打印分隔线（使用=号更整齐）

	// 选项部分
	fmt.Println("\033[1;34m请选择一个操作:\033[0m") // 蓝色
	options := []string{"log", "ps", "start", "stop", "restart", "exec", "images"}
	for i, option := range options {
		// 为选项列表添加颜色
		// 选项序号白色，操作命令加粗的绿色或红色
		var color string
		switch option {
		case "start", "restart":
			color = "\033[1;32m" // 绿色，表示启动或重启
		case "stop":
			color = "\033[1;31m" // 红色，表示停止
		case "exec", "log":
			color = "\033[1;33m" // 黄色，表示交互式命令或日志
		case "ps":
			color = "\033[1;36m" // 青色，表示查看容器状态
		case "images":
			color = "\033[1;35m" // 紫色，表示查看镜像
		}
		// 输出命令行选项，序号加粗黑色，命令名彩色
		fmt.Printf("\033[30m%d.\033[0m %s%s\033[0m\n", i+1, color, option)
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

	var filter string

	// 获取用户输入的过滤条件
	if action == "images" {
		// 对于镜像操作，提示用户输入镜像名称过滤条件
		fmt.Print("请输入镜像名称过滤条件（可选，留空表示不过滤）: ")
		scanner.Scan()
		filter = scanner.Text()
	} else {
		// 对于容器操作，提示用户输入容器名称过滤条件
		fmt.Print("请输入容器名称过滤条件（可选，留空表示不过滤）: ")
		scanner.Scan()
		filter = scanner.Text()
	}

	// 根据选择执行操作
	if action == "images" {
		// 查看镜像列表
		cmd := exec.Command("docker", "images", "--format", "{{.Repository}}|{{.Tag}}|{{.ID}}|{{.CreatedAt}}|{{.Size}}")
		output, err := cmd.Output()
		if err != nil {
			fmt.Println("获取镜像列表失败:", err)
			return
		}

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		var images []string
		for _, line := range lines {
			// 过滤镜像信息
			if filter != "" && !strings.Contains(line, filter) {
				continue
			}
			images = append(images, line)
		}

		// 输出标题行
		fmt.Println("\033[1;30m" + strings.Repeat("-", getTerminalWidth()) + "\033[0m") // 分隔线（深灰色）
		fmt.Printf("\033[1;30m%-50s \033[0m\033[1;34m%-15s \033[0m\033[1;33m%-15s \033[0m\033[1;35m%-20s \033[0m\033[1;32m%-10s\033[0m\n",
			"Repository", "Tag", "ID", "CreatedAt", "Size")

		// 输出镜像列表，应用颜色
		for _, image := range images {
			parts := strings.Split(image, "|")
			// 格式化时间为简洁的日期格式 2024-10-30 22:10:08 +0800 CST
			createdTime, err := time.Parse("2006-01-02 15:04:05 -0700 MST", parts[3])
			if err != nil {
				createdTime = time.Now() // 如果解析错误，使用当前时间
			}
			// 格式化为日期和时间（YYYY-MM-DD HH:MM:SS）
			formattedCreated := createdTime.Format("2006-01-02 15:04:05")

			// 限制列宽最大长度，避免过长内容影响表格格式
			truncate := func(s string, maxLen int) string {
				if len(s) > maxLen {
					return s[:maxLen-3] + "..."
				}
				return s
			}

			// 输出镜像列表，并应用颜色
			if len(parts) == 5 {
				fmt.Printf("\033[30m%-50s \033[0m\033[34m%-15s \033[0m\033[33m%-15s \033[0m\033[35m%-20s \033[0m\033[32m%-10s\033[0m\n",
					truncate(parts[0], 50), truncate(parts[1], 15), truncate(parts[2], 15), formattedCreated, truncate(parts[4], 10))
			}
		}
		return
	}

	// 获取容器列表
	containers := getAllContainers(filter) // 传入过滤条件
	if len(containers) == 0 {
		fmt.Println("没有找到容器")
		return
	}

	// 设置 tabwriter 来美化输出格式

	// 设置颜色
	//titleColor := "\033[1;34m" // 蓝色，标题
	resetColor := "\033[0m"   // 重置颜色
	greenColor := "\033[32m"  // 绿色，表示“Up”状态
	redColor := "\033[31m"    // 红色，表示“Exited”状态
	yellowColor := "\033[33m" // 黄色，表示其他状态

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Println("\033[1;30m" + strings.Repeat("-", getTerminalWidth()) + "\033[0m") // 分隔线（深灰色）
	fmt.Fprintf(w, "N  CONTAINER ID\tNAMES\tSTATUS\tPORTS\tIMAGE\tCREATED\tCOMMAND\n")

	for i, container := range containers {
		// 格式化时间为简洁的日期格式
		createdTime, err := time.Parse("2006-01-02 15:04:05 -0700 MST", container.Created)
		if err != nil {
			createdTime = time.Now() // 如果解析错误，使用当前时间
		}
		// 格式化为日期和时间（YYYY-MM-DD HH:MM:SS）
		formattedCreated := createdTime.Format("2006-01-02 15:04:05")

		// 检查状态并设置颜色
		var statusColor string
		if strings.Contains(container.Status, "Up") {
			statusColor = greenColor // 绿色
		} else if strings.Contains(container.Status, "Exited") {
			statusColor = redColor // 红色
		} else {
			statusColor = yellowColor // 黄色（其他状态）
		}

		// 输出容器信息
		fmt.Fprintf(w, "%d  %s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			i+1, container.ID, container.Name, statusColor+container.Status+resetColor, container.Ports, container.Image, formattedCreated, container.Command)
	}
	w.Flush() // 刷新缓冲区，将内容打印到控制台

	if action == "ps" {
		return
	}

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
		if filter != "" && (!strings.Contains(parts[6], filter) && !strings.Contains(parts[1], filter)) {
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

// 获取终端宽度
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// 如果获取终端宽度失败，默认返回 80
		return 80
	}
	return width
}

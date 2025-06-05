package main

import (
	"bufio"
	"fmt"
	"github.com/duke-git/lancet/v2/strutil"
	"golang.org/x/term"
	"os"
	"os/exec"
	"path/filepath"
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

type ImageInfo struct {
	Repository string
	Tag        string
	ID         string
	CreatedAt  string
	Size       string
}

func main() {
	// 欢迎信息、作者信息、分隔线部分
	fmt.Println("\033[1;32mWelcome to the Service Management Tool v1.0.0\033[0m") // 绿色
	fmt.Println("\033[1;32mAuthor: He Hanpeng\033[0m")                            // 绿色
	fmt.Println("\033[1;32mEmail: hehanpengyy@163.com\033[0m")                    // 绿色
	fmt.Println(strings.Repeat("=", getTerminalWidth()))                          // 打印分隔线（使用=号更整齐）

	// 选项部分
	fmt.Println("\033[1;34m请选择一个操作:\033[0m") // 蓝色
	options := []string{"log", "ps", "start", "stop", "restart", "up", "exec", "images", "image-save", "image-prune"}
	for i, option := range options {
		// 为选项列表添加颜色
		var color string
		switch option {
		case "start", "restart", "up":
			color = "\033[1;32m" // 绿色，表示启动或重启
		case "stop":
			color = "\033[1;31m" // 红色，表示停止
		case "exec", "log":
			color = "\033[1;33m" // 黄色，表示交互式命令或日志
		case "ps":
			color = "\033[1;36m" // 青色，表示查看容器状态
		case "images":
			color = "\033[1;34m" // 蓝色，表示查看镜像
		case "image-save":
			color = "\033[1;35m" // 紫色，表示保存镜像
		case "image-prune":
			color = "\033[1;31m" // 红色，表示清理镜像
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

	// 特殊处理
	if action == "image-prune" {
		fmt.Print("\033[31m正在执行悬空镜像清理...\033[0m\n")
		executeActionCommandWithInteractive(action, ContainerInfo{})
		return
	}

	var filter string

	// 获取用户输入的过滤条件
	if strings.Contains(action, "image") {
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

	if strings.Contains(action, "image") {
		// 查看镜像列表
		images := getAllImageList(filter)
		if len(images) == 0 {
			fmt.Println("没有找到镜像")
			return
		}

		// 输出标题行
		fmt.Println("\033[1;30m" + strings.Repeat("-", getTerminalWidth()) + "\033[0m") // 分隔线（深灰色）
		fmt.Printf("\033[30m%-3s\033[0m \033[1;30m%-50s \033[0m\033[1;34m%-30s \033[0m\033[1;33m%-15s \033[0m\033[1;35m%-20s \033[0m\033[1;32m%-15s\033[0m\n",
			"N", "Repository", "Tag", "ID", "CreatedAt", "Size")

		// 输出镜像列表，应用颜色
		for i, image := range images {
			// 格式化时间为简洁的日期格式 2024-10-30 22:10:08 +0800 CST
			createdTime, err := time.Parse("2006-01-02 15:04:05 -0700 MST", image.CreatedAt)
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
			fmt.Printf("\033[30m%-3d\033[0m \033[30m%-50s \033[0m\033[34m%-30s \033[0m\033[33m%-15s \033[0m\033[35m%-20s \033[0m\033[32m%-10s\033[0m\n",
				i+1, truncate(image.Repository, 50), truncate(image.Tag, 30), truncate(image.ID, 15), formattedCreated, truncate(image.Size, 15))
		}

		if action == "image-save" {
			// 镜像操作
			fmt.Print("请输入镜像序号: ")
			scanner.Scan()
			imageIndex, _ := strconv.Atoi(scanner.Text())
			if len(images) > 1 {
				if imageIndex < 1 || imageIndex > len(images) {
					fmt.Println("无效选择")
					return
				}
			} else {
				imageIndex = 1
			}
			image := images[imageIndex-1]
			fmt.Printf("当前选择镜像：%s:%s\n", image.Repository, image.Tag)
			// 镜像操作
			fmt.Print("请输入文件名（可选，留空表示默认）: ")
			scanner.Scan()
			finaFileName := scanner.Text()
			if len(finaFileName) == 0 {
				// 获取当前时间
				//startTime := time.Now()
				// 格式化当前时间为字符串，作为文件名
				// 格式：YYYY-MM-DD_HH-MM-SS 后缀
				//suffix := startTime.Format("2006-01-02_15-04-05")
				// 导出文件名 为镜像名称+tag+id+时间戳
				tempArr := strings.Split(image.Repository, "/")
				prefix := tempArr[len(tempArr)-1]
				finaFileName = fmt.Sprintf("%s:%s-%s.tar", prefix, strings.ReplaceAll(image.Tag, "/", "-"), strings.ReplaceAll(image.ID, "/", "-"))
			} else {
				finaFileName = fmt.Sprintf("%s.tar", finaFileName)
			}
			_, err := os.Stat(finaFileName)
			if err == nil {
				fmt.Println("文件已存在，是否覆盖？(y/n)")
				scanner.Scan()
				choice := scanner.Text()
				if choice == "n" {
					fmt.Println("操作已取消。")
					return
				}
			}
			fmt.Printf("导出文件名: %s\n", finaFileName)
			executeCommandWithInteractive(action, "docker", "image", "save", "-o", finaFileName, fmt.Sprintf("%s:%s", image.Repository, image.Tag))
			//// 检查目标文件是否存在
			//newName := fmt.Sprintf("%s:%s-%s.tar", image.Repository, image.Tag, image.ID)
			//// 重命名文件
			//err := fileutil.CopyFile(newName, fileName)
			////err := os.Rename(fileName, newName)
			//if err != nil {
			//	// 如果重命名失败，删除源文件
			//	fmt.Println("Error renaming file:", err)
			//	fmt.Println("Attempting to delete the source file:", fileName)
			//	// 尝试删除源文件
			//	removeErr := os.Remove(fileName)
			//	if removeErr != nil {
			//		fmt.Println("Error deleting source file:", removeErr)
			//	} else {
			//		fmt.Println("Source file deleted successfully.")
			//	}
			//	return
			//}
			fmt.Println("File save successfully.")
			//fmt.Printf("导出文件名: %s\n", newName)
			return
		}
		return
	} else if !strings.Contains(action, "image") {
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
		fmt.Fprintf(w, "N  CONTAINER ID\tNAMES\tSTATUS\tIMAGE\tCREATED\tCOMMAND\tPORTS\n")

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
				i+1, container.ID, container.Name, statusColor+container.Status+resetColor, container.Image, formattedCreated, container.Command, container.Ports)
		}
		w.Flush() // 刷新缓冲区，将内容打印到控制台

		if action == "ps" {
			return
		}

		// 容器操作部分
		fmt.Print("请输入容器序号: ")
		scanner.Scan()
		choose := scanner.Text()
		if choose == "" {
			choose = "1"
		}
		containerIndex, _ := strconv.Atoi(choose)
		if containerIndex < 1 || containerIndex > len(containers) {
			fmt.Println("无效选择")
			return
		}

		container := containers[containerIndex-1]
		executeActionCommandWithInteractive(action, container)
		return
	} else {
		fmt.Println("未知操作")
		return
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

func getAllImageList(filter string) []ImageInfo {
	// 查看镜像列表
	cmd := exec.Command("docker", "images", "--format", "{{.Repository}}|{{.Tag}}|{{.ID}}|{{.CreatedAt}}|{{.Size}}")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("获取镜像列表失败:", err)
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var imageList []ImageInfo
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}

		// 过滤镜像信息
		if filter != "" && (!strings.Contains(parts[0], filter) && !strings.Contains(parts[1], filter)) {
			continue
		}
		image := ImageInfo{
			Repository: parts[0],
			Tag:        parts[1],
			ID:         parts[2],
			CreatedAt:  parts[3],
			Size:       parts[4],
		}
		imageList = append(imageList, image)
	}
	return imageList
}

func executeActionCommandWithInteractive(action string, container ContainerInfo) {
	var cmd *exec.Cmd
	switch action {
	case "start":
		cmd = exec.Command("docker", "start", container.ID)
	case "stop":
		cmd = exec.Command("docker", "stop", container.ID)
	case "restart":
		cmd = exec.Command("docker", "restart", container.ID)
	case "up":
		// 执行 docker compose
		// 判断当前文件夹名
		// 获取当前工作目录
		dir, err := os.Getwd()
		if err != nil {
			fmt.Println("获取当前目录失败:", err)
			return
		}
		// 获取当前目录名
		dirName := filepath.Base(dir)
		fmt.Println("当前目录名:", dirName)
		// 定义 docker-compose 文件路径
		filePath := "docker-compose.yml"
		// 使用 os.Stat 检查文件是否存在
		_, err = os.Stat(filePath)
		if err != nil {
			// 如果文件不存在
			if os.IsNotExist(err) {
				fmt.Println("文件不存在:", filePath)
			} else {
				// 其他错误
				fmt.Println("检查文件时发生错误:", err)
			}
			return
		}
		newCompose := false
		upName := strutil.BeforeLast(strutil.After(container.Name, dirName+"_"), "_")
		if upName == "" || container.Name == upName {
			newCompose = true
			upName = strutil.BeforeLast(strutil.After(container.Name, dirName+"-"), "-")
			if upName == "" || container.Name == upName {
				fmt.Println("无法解析模块名,请手工执行命令")
				return
			}
		}
		// 打印模块名
		fmt.Println("模块名:", upName)
		if newCompose {
			cmd = exec.Command("docker", "compose", "up", "-d", upName)
		} else {
			cmd = exec.Command("docker-compose", "up", "-d", upName)
		}
	case "exec":
		arg := "/bin/bash"
		// 使用 `-it` 伪终端参数
		if strings.Contains(container.Image, "alpine") {
			arg = "/bin/sh"
		}
		cmd = exec.Command("docker", "exec", "-it", container.ID, arg)
	case "log":
		cmd = exec.Command("docker", "logs", "--tail", "1000", "-f", container.ID)
	case "image-prune":
		cmd = exec.Command("docker", "image", "prune")
	default:
		fmt.Println("未知操作")
		return
	}

	// 将标准输入、输出和错误与当前终端关联
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// 打印最终命令
	fmt.Printf("Command to be executed: %s %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "))
	if err := cmd.Run(); err != nil {
		fmt.Printf("执行 %s 命令失败: %v\n", action, err)
	} else {
		fmt.Printf("%s 操作成功执行\n", action)
	}
}

func executeCommandWithInteractive(action, name string, arg ...string) {
	var cmd *exec.Cmd
	cmd = exec.Command(name, arg...)
	// 将标准输入、输出和错误与当前终端关联
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// 打印最终命令
	fmt.Printf("Command to be executed: %s %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "))
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

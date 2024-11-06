#!/bin/bash

# 配置文件路径
CONFIG_FILE="config-sh.yaml"

# 操作选择
echo "Available operations:"
operations=("start" "stop" "up" "restart")
for i in "${!operations[@]}"; do
    echo "$((i + 1)): ${operations[i]}"
done

# 提示用户选择操作
read -p "Enter the operation number (1-${#operations[@]}): " operation_number

# 验证操作编号
if ! [[ "$operation_number" =~ ^[1-4]$ ]]; then
    echo "Invalid operation number."
    exit 1
fi

# 获取操作名称
operation="${operations[$((operation_number - 1))]}"

# 读取服务器列表
echo ""
echo "Available servers:"
server_names=($(yq eval 'keys | .[]' "$CONFIG_FILE"))

# 计算服务器数量
server_count=${#server_names[@]}

# 检查有效服务器
if [[ "$server_count" -eq 0 ]]; then
    echo "No servers found in the YAML configuration. Please check your config.yaml."
    exit 1
fi

# 打印服务器信息
for index in "${!server_names[@]}"; do
    remark=$(yq eval ".\"${server_names[$index]}\".remark" "$CONFIG_FILE")
    echo "$((index + 1))) ${server_names[$index]} ($remark)"
done

# 提示用户选择服务器编号
read -p "Enter the server number to run (1-$server_count): " server_number

# 验证服务器编号
if ! [[ "$server_number" =~ ^[1-$server_count]$ ]]; then
    echo "Invalid server number."
    exit 1
fi

# 获取服务器配置
server_name=${server_names[$((server_number - 1))]}
ssh_user=$(yq eval ".\"$server_name\".sshUser" "$CONFIG_FILE")
ssh_host=$(yq eval ".\"$server_name\".sshHost" "$CONFIG_FILE")
ssh_port=$(yq eval ".\"$server_name\".sshPort" "$CONFIG_FILE")
ssh_password=$(yq eval ".\"$server_name\".sshPassword" "$CONFIG_FILE")
path=$(yq eval ".\"$server_name\".path" "$CONFIG_FILE")
services=($(yq eval ".\"$server_name\".serviceName[]" "$CONFIG_FILE"))

# 打印服务器配置
echo ""
echo "Configuration for server $server_name:"
echo "SSH User: $ssh_user"
echo "SSH Host: $ssh_host"
echo "SSH Port: $ssh_port"
echo "SSH Password: $ssh_password"
echo "Path: $path"
echo "Services:"

if [ ${#services[@]} -eq 0 ]; then
    echo "  None"
else
    for service in "${services[@]}"; do
        echo "  - $service"
    done
fi

# 提示用户选择服务模式
echo ""
echo "Select service selection mode:"
echo "1) single (choose one service)"
echo "2) multi (choose multiple services)"
read -p "Enter the mode (1-2): " mode_selection

# 服务选择处理
case "$mode_selection" in
    1)
        echo "Available services for server $server_name:"
        echo "0) all"
        for index in "${!services[@]}"; do
            echo "$((index + 1))) ${services[$index]}"
        done

        read -p "Enter the service number to run (0-${#services[@]}): " service_number
        if [[ "$service_number" -eq 0 ]]; then
            a=""  # 设置为空字符串
        elif [[ "$service_number" -gt 0 && "$service_number" -le "${#services[@]}" ]]; then
            a="${services[$((service_number - 1))]}"
        else
            echo "Invalid service number."
            exit 1
        fi
        ;;
    2)
        echo "Available services for server $server_name:"
        for index in "${!services[@]}"; do
            echo "$((index + 1))) ${services[$index]}"
        done

        read -p "Enter the service number(s) to run, separated by space (1 to ${#services[@]}): " -a multi_services_input
        valid_services=()

        for input in "${multi_services_input[@]}"; do
            if [[ "$input" =~ ^[1-9][0-9]*$ ]] && [ "$input" -le "${#services[@]}" ]; then
                valid_services+=("${services[$((input - 1))]}")
            else
                echo "Invalid service number: $input"
            fi
        done

        if [ ${#valid_services[@]} -gt 0 ]; then
            a=$(IFS=' '; echo "${valid_services[*]}")
        else
            echo "No valid service names provided. Exiting."
            exit 1
        fi
        ;;
    *)
        echo "Invalid selection mode."
        exit 1
        ;;
esac

# 提示最终确认
echo "You selected operation: $operation on server: $server_name for services: $a"
read -p "Do you want to proceed? (y/n): " confirm

if [[ "$confirm" != "y" ]]; then
    echo "Operation cancelled."
    exit 0
fi

TASK_NAME="${operation}-docker"  # 生成任务名称

# 执行任务
echo "Running task: $operation on $server_name for services: $a"
task "$TASK_NAME" SSH_USER="$ssh_user" SSH_HOST="$ssh_host" SSH_PORT="$ssh_port" SSH_PASSWORD="$ssh_password" DOCKER_COMPOSE_PATH="$path" SERVICE_NAME="$a"

# 检查任务执行结果
if [[ $? -eq 0 ]]; then
    echo "Task \"$operation\" executed successfully on server \"$server_name\" for services: $a."
else
    echo "Failed to run task \"$operation\": exit status $?"
    exit 1
fi

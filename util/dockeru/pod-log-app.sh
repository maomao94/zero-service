#!/bin/bash

Pod_Namespace=starworld

kubectl get pod -n $Pod_Namespace | grep Running | awk '{print NR " " $1}'
kubectl get pod -n $Pod_Namespace | grep Running | awk '{print NR " " $1}' > /tmp/pod_list

if [ `kubectl get pod -n $Pod_Namespace | grep Running | wc -l` -eq 0 ]
then
  echo "没有pod运行"
  exit
fi

read -p "请选择需要查看日志的pod: " k8s1

# 获取选中的 pod 名称
pod_name=$(cat /tmp/pod_list | head -$k8s1 | tail -1 | awk '{print $2}')

# 获取容器名（假设一个 pod 中只有一个容器）
container_name=$(kubectl get pod $pod_name -n $Pod_Namespace -o jsonpath='{.spec.containers[0].name}')

# 使用 kubectl logs -f 显示日志
kubectl logs -f $pod_name --tail=100 -n $Pod_Namespace -c $container_name
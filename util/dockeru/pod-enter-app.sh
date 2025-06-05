#!/bin/bash

Pod_Namespace=starworld

kubectl get pod -n $Pod_Namespace | grep Running | awk '{print NR " " $1}'
kubectl get pod -n $Pod_Namespace | grep Running | awk '{print NR " " $1}' > /tmp/pod_list

if [ `kubectl get pod -n $Pod_Namespace | grep Running | wc -l` -eq 0 ]
then
  echo "没有pod运行"
  exit
fi

read -p "请选择需要进入的pod: " k8s1
#kubectl exec -it `cat /tmp/pod_list | head -$k8s1 | tail -1 | awk '{print $2}'` bash -n dev-evcard-md -c `cat /tmp/pod_list | head -$k8s1 | tail -1 | sed 's/-[0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z]-[0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z]//g' | sed 's/-[0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z]-[0-9a-z][0-9a-z][0-9a-z][0-9a-z][0-9a-z]//g' | awk '{print $2}'`
kubectl exec -it `cat /tmp/pod_list | head -$k8s1 | tail -1 | awk '{print $2}'` bash -n $Pod_Namespace                                                                                                                                                                                                                                        

package utils

import (
	"MBCTG/pkg/definition"
	"context"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Contains 判断字符串 slice 是否包含指定字符串
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetNodeIPByName 根据节点名称返回对应 IP；找不到则返回错误信息字符串
func GetNodeIPByName(name string) string {
	if ip, ok := definition.NodeIps[name]; ok {
		return ip
	}
	return "node名称错误"
}

// GetNodeNameByIP 根据 IP 查找对应节点名称
func GetNodeNameByIP(ip string) string {
	for nodeName, nodeIP := range definition.NodeIps {
		if nodeIP == ip {
			return nodeName
		}
	}
	return "ip错误"
}

// K8sNodesAvailable 返回满足条件的可用节点；当 isCloud 为 true 时，仅返回云节点（需包含 label role=cloud 且名称在 CLOUD_NODES 中）
func K8sNodesAvailable(isCloud bool) ([]*corev1.Node, error) {
	nodesList, err := definition.ClientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var readyNodes []*corev1.Node
	for i := range nodesList.Items {
		node := &nodesList.Items[i]
		// 跳过 unschedulable 的节点
		if node.Spec.Unschedulable {
			continue
		}
		// 遍历节点条件，查找 Ready 条件为 True 的情况
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				// 若只筛选云节点，则需满足 label "role" == "cloud" 且节点名称在 config.CLOUD_NODES 中
				if isCloud {
					if role, ok := node.Labels["role"]; ok && role == "cloud" && Contains(definition.CloudNodes, node.Name) {
						readyNodes = append(readyNodes, node)
						break // 找到符合条件的 Ready 状态，跳出循环
					}
				} else {
					readyNodes = append(readyNodes, node)
					break
				}
			}
		}
	}
	return readyNodes, nil
}

// K8sNodesAvailableNames 返回所有满足条件的节点名称列表
func K8sNodesAvailableNames(isCloud bool) ([]string, error) {
	nodes, err := K8sNodesAvailable(isCloud)
	if err != nil {
		return nil, err
	}
	if nodes == nil || len(nodes) == 0 {
		return nil, errors.New("没有符合条件的节点")
	}
	var names []string
	for _, node := range nodes {
		names = append(names, node.Name)
	}
	return names, nil
}

// GetK8sNodeByName 根据名称查找 k8s Node 对象；找不到时返回错误
func GetK8sNodeByName(name string) (*corev1.Node, error) {
	nodes, err := K8sNodesAvailable(true) // 仅筛选云节点
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		if node.Name == name {
			return node, nil
		}
	}
	return nil, errors.New("name错误")
}

// GetK8sPodMemoryRequest 获取 Pod 所有容器内存请求总和（字节）
func GetK8sPodMemoryRequest(pod *corev1.Pod) float64 {
	var sum float64 = 0
	for _, container := range pod.Spec.Containers {
		if memQty, exists := container.Resources.Requests[corev1.ResourceMemory]; exists {
			sum += float64(memQty.Value())
		}
	}
	return sum
}

// GetK8sPodMemoryLimits 获取 Pod 所有容器内存限制总和（字节）
func GetK8sPodMemoryLimits(pod *corev1.Pod) float64 {
	var sum float64 = 0
	for _, container := range pod.Spec.Containers {
		if memQty, exists := container.Resources.Limits[corev1.ResourceMemory]; exists {
			sum += float64(memQty.Value())
		}
	}
	return sum
}

// GetK8sPodCpuRequest 获取 Pod 所有容器 CPU 请求总和（毫核）
func GetK8sPodCpuRequest(pod *corev1.Pod) float64 {
	var sum float64 = 0
	for _, container := range pod.Spec.Containers {
		if cpuQty, exists := container.Resources.Requests[corev1.ResourceCPU]; exists {
			sum += float64(cpuQty.MilliValue())
		}
	}
	return sum
}

// GetK8sPodCpuLimits 获取 Pod 所有容器 CPU 限制总和（毫核）
func GetK8sPodCpuLimits(pod *corev1.Pod) float64 {
	var sum float64 = 0
	for _, container := range pod.Spec.Containers {
		if cpuQty, exists := container.Resources.Limits[corev1.ResourceCPU]; exists {
			sum += float64(cpuQty.MilliValue())
		}
	}
	return sum
}

// GetNodePods 获取所有云节点上 namespace 为 "k8s" 且状态为 Running 的 Pod，并转换为自定义 Pod 对象
func GetNodePods() (map[string][]*definition.Pod, error) {
	nodes, err := K8sNodesAvailable(true)
	if err != nil {
		return nil, err
	}
	nodePods := make(map[string][]*definition.Pod)
	for _, node := range nodes {
		// 根据节点名称筛选 Pod
		fieldSelector := fmt.Sprintf("spec.nodeName=%s", node.Name)
		podsList, err := definition.ClientSet.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
			FieldSelector: fieldSelector,
		})
		if err != nil {
			return nil, err
		}
		var pods []*definition.Pod
		for _, pod := range podsList.Items {
			if pod.Namespace == "k8s" && pod.Status.Phase == corev1.PodRunning {
				pods = append(pods, ConvertK8sPodToMyPod(&pod))
			}
		}
		nodePods[node.Name] = pods
	}
	return nodePods, nil
}

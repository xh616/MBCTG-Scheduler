package definition

import corev1 "k8s.io/api/core/v1"

type Node struct {
	IP                string      // 节点 IP
	Name              string      // 节点名称
	K8sNode           interface{} // k8s_node 对象，可以根据需要替换为具体类型
	CapacityCPU       float64     // 节点 CPU 总量
	AllocatableCPU    float64     // 节点 CPU 可用量
	CapacityMemory    float64     // 节点内存总容量
	AllocatableMemory float64     // 节点内存可用容量
}

type Pod struct {
	Name          string      // Pod 名称
	Node          string      // Pod 所在节点
	K8sPod        interface{} // k8s 的 Pod 对象，可替换为具体类型
	MemoryRequest float64     // Pod 的内存请求
	CPURequest    float64     // Pod 的 CPU 请求
	MemoryLimits  float64     // Pod 的内存限制
	CPULimits     float64     // Pod 的 CPU 限制
}

// NewPod 构造函数
func NewPod(name string, node string, k8sPod *corev1.Pod, memoryRequest, cpuRequest, memoryLimits, cpuLimits float64) *Pod {
	return &Pod{
		Name:          name,
		Node:          node,
		K8sPod:        k8sPod,
		MemoryRequest: memoryRequest,
		CPURequest:    cpuRequest,
		MemoryLimits:  memoryLimits,
		CPULimits:     cpuLimits,
	}
}

func NewNode(ip string, name string, k8sNode *corev1.Node, capacityCpu, allocatableCpu, capacityMemory, allocatableMemory float64) *Node {
	return &Node{
		IP:                ip,
		Name:              name,
		K8sNode:           k8sNode,
		CapacityCPU:       capacityCpu,
		AllocatableCPU:    allocatableCpu,
		CapacityMemory:    capacityMemory,
		AllocatableMemory: allocatableMemory,
	}
}

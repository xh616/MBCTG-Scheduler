package utils

import (
	"MBCTG/pkg/definition"
	"errors"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
)

// memConvertToInt 将内存资源字符串（例如 "2Gi"）转换为字节数2147483648
func memConvertToInt(resourceString string) (float64, error) {
	if strings.Contains(resourceString, "Ki") {
		valStr := strings.TrimSuffix(resourceString, "Ki") //如果 `s` 以 `suffix` 结尾，则去除该后缀。
		val, err := strconv.ParseInt(valStr, 10, 64)       // 转十进制int
		if err != nil {
			return 0, err
		}
		return float64(val * 1 << 10), nil
	} else if strings.Contains(resourceString, "Mi") {
		valStr := strings.TrimSuffix(resourceString, "Mi")
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return 0, err
		}
		return float64(val * 1 << 20), nil
	} else if strings.Contains(resourceString, "Gi") {
		valStr := strings.TrimSuffix(resourceString, "Gi")
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return 0, err
		}
		return float64(val * 1 << 30), nil
	}
	return 0, errors.New("不支持的内存格式: " + resourceString)
}

// cpuConvertToMilliValue 将 CPU 资源字符串转换为毫核数
// 例如： "2" --> 2000， "250m" --> 250
func cpuConvertToMilliValue(resourceString string) (float64, error) {
	if strings.Contains(resourceString, "m") {
		valStr := strings.TrimSuffix(resourceString, "m")
		return strconv.ParseFloat(valStr, 64)
	} else {
		val, err := strconv.ParseFloat(resourceString, 64)
		if err != nil {
			return 0, err
		}
		return val * 1000, nil
	}
}

// ConvertK8sPodToMyPod 将 Kubernetes Pod 对象转换为自定义 Pod 对象
func ConvertK8sPodToMyPod(k8sPod *corev1.Pod) *definition.Pod {
	memReq := GetK8sPodMemoryRequest(k8sPod)
	cpuReq := GetK8sPodCpuRequest(k8sPod)
	memLimits := GetK8sPodMemoryLimits(k8sPod)
	cpuLimits := GetK8sPodCpuLimits(k8sPod)

	return definition.NewPod(k8sPod.ObjectMeta.Name, k8sPod.Spec.NodeName, k8sPod, memReq, cpuReq, memLimits, cpuLimits)
}

// ConvertAllK8sNodesToMyNodes 所有k8s的node对象转换为我的Node对象
func ConvertAllK8sNodesToMyNodes() (map[string]*definition.Node, error) {
	nodes, err := K8sNodesAvailable(true)
	if err != nil {
		return nil, err
	}

	myNodes := make(map[string]*definition.Node)
	for _, n := range nodes {
		if len(n.Status.Addresses) == 0 {
			continue
		}
		// 先将 map 中的 Quantity 复制到局部变量中
		cpuCapQuantity := n.Status.Capacity[corev1.ResourceCPU]
		cpuAllocQuantity := n.Status.Allocatable[corev1.ResourceCPU]
		memCapQuantity := n.Status.Capacity[corev1.ResourceMemory]
		memAllocQuantity := n.Status.Allocatable[corev1.ResourceMemory]

		// 再对局部变量取地址调用 String() 方法
		cpuCapacityStr := (&cpuCapQuantity).String()
		cpuAllocStr := (&cpuAllocQuantity).String()
		memCapacityStr := (&memCapQuantity).String()
		memAllocStr := (&memAllocQuantity).String()

		cpuCapacity, err := cpuConvertToMilliValue(cpuCapacityStr)
		if err != nil {
			return nil, err
		}
		cpuAlloc, err := cpuConvertToMilliValue(cpuAllocStr)
		if err != nil {
			return nil, err
		}
		memCapacity, err := memConvertToInt(memCapacityStr)
		if err != nil {
			return nil, err
		}
		memAlloc, err := memConvertToInt(memAllocStr)
		if err != nil {
			return nil, err
		}

		myNodes[n.ObjectMeta.Name] = definition.NewNode(
			n.Status.Addresses[0].Address,
			n.ObjectMeta.Name,
			n,
			cpuCapacity,
			cpuAlloc,
			memCapacity,
			memAlloc,
		)
	}
	return myNodes, nil
}

package definition

import (
	"MBCTG/utils"
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type CustomScheduler struct {
	Clientset     *kubernetes.Clientset // 用于调用 k8s API
	K8sNodes      []*corev1.Node        // k8s 节点对象集合（云节点）
	K8sNodesName  []string              // k8s 节点名称集合
	MyNodes       map[string]*Node      // 转换后的自定义 Node 对象，key 为节点名称
	NodePods      map[string][]*Pod     // 每个节点上已有 Pod 的集合
	SchedulerName string                // 调度器名称
}

// NewCustomScheduler 创建 CustomScheduler 实例
func NewCustomScheduler(schedulerName string) (*CustomScheduler, error) {
	// 获取集群配置
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	// 若未传入调度器名称，则使用默认值（可从配置中读取）
	if schedulerName == "" {
		schedulerName = SchedulerName
	}
	// 获取 k8s 节点对象集合和名称集合
	k8sNodes, err := utils.K8sNodesAvailable(true)
	if err != nil {
		return nil, err
	}
	k8sNodesName, err := utils.K8sNodesAvailableNames(true)
	if err != nil {
		return nil, err
	}
	// 将 k8s 节点转换为自定义 Node 对象
	nodes, err := utils.ConvertAllK8sNodesToMyNodes()
	if err != nil {
		return nil, err
	}
	// 获取每个节点上已有 Pod 的信息
	nodePods, err := utils.GetNodePods()
	if err != nil {
		return nil, err
	}

	return &CustomScheduler{
		Clientset:     cs,
		K8sNodes:      k8sNodes,
		K8sNodesName:  k8sNodesName,
		MyNodes:       nodes,
		NodePods:      nodePods,
		SchedulerName: schedulerName,
	}, nil
}

// Schedule 根据传入的 k8sPod 进行调度
func (cs *CustomScheduler) Schedule(k8sPod *corev1.Pod) error {
	fmt.Printf("---->调度pod: %s <----\n", k8sPod.ObjectMeta.Name)
	// 转换 k8sPod 为自定义 Pod 对象
	t0 := utils.ConvertK8sPodToMyPod(k8sPod)
	// 选择合适的节点
	chosenNode := cs.NonCooperativeGame(t0)
	if chosenNode == nil {
		return fmt.Errorf("未找到满足资源需求的节点")
	}
	// 根据选择的节点名称从自定义 MyNodes 中获取节点对象
	customNode, ok := cs.MyNodes[chosenNode.ObjectMeta.Name]
	if !ok {
		return fmt.Errorf("自定义节点中未找到: %s", chosenNode.ObjectMeta.Name)
	}
	// 绑定并部署 Pod 到选定节点
	cs.placePod(k8sPod, customNode)
	// 可选：等待一段时间后评价调度结果
	// time.Sleep(5 * time.Second)
	cs.judge()
	return nil
}

// NonCooperativeGame 实现非合作博弈调度算法，返回选择的 k8s 节点对象
func (cs *CustomScheduler) NonCooperativeGame(t0 *Pod) *corev1.Node {
	var chosenNode *corev1.Node
	nodesCPU, err := utils.HttpGetNodeMonitor("cpu")
	if err != nil {
		fmt.Println("获取节点 CPU 监控数据错误:", err)
		return nil
	}
	nodesMem, err := utils.HttpGetNodeMonitor("mem")
	if err != nil {
		fmt.Println("获取节点内存监控数据错误:", err)
		return nil
	}
	cpuLeft := make(map[string]int64)
	memLeft := make(map[string]int64)
	var maxUtility float64 = 0
	// 遍历所有 k8s 节点
	for _, n := range cs.K8sNodes {
		customNode, ok := cs.MyNodes[n.ObjectMeta.Name]
		if !ok {
			continue
		}
		// nodesCPU 与 nodesMem 的 key 为节点名称，单位分别为毫核和字节
		cpuUsed, cpuOk := nodesCPU[n.ObjectMeta.Name]
		memUsed, memOk := nodesMem[n.ObjectMeta.Name]
		if !cpuOk || !memOk {
			continue
		}
		cpuLeft[customNode.Name] = customNode.CapacityCPU - int64(cpuUsed)
		memLeft[customNode.Name] = customNode.CapacityMemory - int64(memUsed)
		// 判断节点是否能容纳待调度 Pod
		if t0.CPURequest < cpuLeft[customNode.Name] && t0.MemoryRequest < memLeft[customNode.Name] {
			utility := float64(t0.CPURequest)/float64(cpuLeft[customNode.Name]) + float64(t0.MemoryRequest)/float64(memLeft[customNode.Name])
			if utility > maxUtility {
				maxUtility = utility
				chosenNode = n
			}
		}
	}
	return chosenNode
}

// judge 打印当前节点的监控数据
func (cs *CustomScheduler) judge() {
	nodesCPU, err := utils.HttpGetNodeMonitor("cpu")
	if err != nil {
		fmt.Println("judge 获取 CPU 数据错误:", err)
	}
	nodesMem, err := utils.HttpGetNodeMonitor("mem")
	if err != nil {
		fmt.Println("judge 获取内存数据错误:", err)
	}
	fmt.Printf("CPU: %v\n内存: %v\n", nodesCPU, nodesMem)
}

// bind 调用 k8s API 将 Pod 绑定到指定节点
func (cs *CustomScheduler) bind(k8sPod *corev1.Pod, nodeName string) error {
	namespace := k8sPod.ObjectMeta.Namespace
	if namespace == "" {
		namespace = "default"
	}
	// 构造目标 Node 对象引用
	target := corev1.ObjectReference{
		APIVersion: "v1",
		Kind:       "Node",
		Name:       nodeName,
	}
	// 构造 binding 对象，metadata 中设置 Pod 名称
	meta := metav1.ObjectMeta{
		Name: k8sPod.ObjectMeta.Name,
	}
	binding := &corev1.Binding{
		Target:     target,
		ObjectMeta: meta,
	}
	fmt.Printf("---->绑定pod: %s 到节点: %s <----\n", k8sPod.ObjectMeta.Name, nodeName)
	// 使用 Pods 接口的 Bind 方法进行绑定
	err := cs.Clientset.CoreV1().Pods(namespace).Bind(context.TODO(), binding, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("调用 CoreV1 API 创建 pod binding 时出错: %v\n", err)
		return err
	}
	fmt.Println("绑定成功")
	return nil
}

// placePod 调用 bind 并将 Pod 添加到 NodePods 中
func (cs *CustomScheduler) placePod(k8sPod *corev1.Pod, node *Node) {
	if err := cs.bind(k8sPod, node.Name); err != nil {
		return
	}
	pod := utils.ConvertK8sPodToMyPod(k8sPod)
	cs.NodePods[node.Name] = append(cs.NodePods[node.Name], pod)
}

// UpdateNodePods 删除指定 Pod 的记录（用于 Pod 删除更新）
func (cs *CustomScheduler) UpdateNodePods(k8sPod *corev1.Pod) {
	removedPod := utils.ConvertK8sPodToMyPod(k8sPod)
	fmt.Printf("---->删除pod: %s <----\n", removedPod.Name)
	plist, ok := cs.NodePods[removedPod.Node]
	if !ok {
		return
	}
	var newList []*Pod
	for _, p := range plist {
		if p.Name != removedPod.Name {
			newList = append(newList, p)
		}
	}
	cs.NodePods[removedPod.Node] = newList
}

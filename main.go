package main

import (
	"MBCTG/pkg"
	"MBCTG/pkg/definition"
	"MBCTG/pkg/utils"
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watchapi "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sync"
	"time"
)

const (
	podQueueSize      = 1000             // Pod队列缓冲区大小
	monitorInterval   = 30 * time.Second // 监控间隔
	schedulerInterval = 30 * time.Second //打印调度器指标间隔
)

var (
	podQueue chan *corev1.Pod  // 带缓冲的Pod队列
	metrics  *SchedulerMetrics // 调度器指标
)

// SchedulerMetrics 调度器性能指标
type SchedulerMetrics struct {
	sync.Mutex
	TotalPodsScheduled int
	FailedSchedules    int
	ActiveSchedules    int
	QueueLength        int
}

// NewSchedulerMetrics 创建新的指标收集器
func NewSchedulerMetrics() *SchedulerMetrics {
	return &SchedulerMetrics{}
}

func main() {
	// 初始化全局变量
	podQueue = make(chan *corev1.Pod, podQueueSize)
	metrics = NewSchedulerMetrics()

	// 初始化Kubernetes客户端
	clientset, err := initKubernetesClient("kube/config")
	if err != nil {
		fmt.Printf("初始化Kubernetes客户端失败: %v\n", err)
		return
	}
	definition.ClientSet = clientset

	// 创建调度器实例
	scheduler, err := pkg.NewCustomScheduler(definition.SchedulerName)
	if err != nil {
		fmt.Printf("创建调度器失败: %v\n", err)
		return
	}

	// 初始化节点信息
	if err := initNodeInfo(); err != nil {
		fmt.Printf("初始化节点信息失败: %v\n", err)
		return
	}

	// 启动监控goroutine
	go monitorClusterResources()
	go printMetrics()

	fmt.Println("---->自定义调度器启动<---->")
	go podScheduler(scheduler)
	// 开始监听Kubernetes事件
	watchK8sEvents(scheduler)
}

// initKubernetesClient 初始化Kubernetes客户端
func initKubernetesClient(kubeconfig string) (*kubernetes.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("加载kubeconfig错误: %v", err)
	}
	return kubernetes.NewForConfig(cfg)
}

// initNodeInfo 初始化节点信息
func initNodeInfo() error {
	readyNodes := make(map[string]string)
	nodesNames, err := utils.K8sNodesAvailableNames(true)
	if err != nil {
		return fmt.Errorf("获取节点名称错误: %v", err)
	}

	for _, n := range nodesNames {
		ip := utils.GetNodeIPByName(n)
		readyNodes[n] = ip
	}
	fmt.Println("可用节点:", readyNodes)

	// 获取基础资源占用
	cpuMonitor, err := utils.HttpGetNodeMonitor("cpu")
	if err != nil {
		return fmt.Errorf("获取CPU监控数据错误: %v", err)
	}
	memMonitor, err := utils.HttpGetNodeMonitor("mem")
	if err != nil {
		return fmt.Errorf("获取Mem监控数据错误: %v", err)
	}

	definition.BasicOccupationCpu = cpuMonitor
	definition.BasicOccupationMem = memMonitor

	fmt.Println("集群初始资源占用:")
	_ = utils.PrintNodeMonitorToRead("cpu")
	_ = utils.PrintNodeMonitorToRead("mem")

	return nil
}

func podScheduler(scheduler *pkg.CustomScheduler) {
	for pod := range podQueue {
		fmt.Printf("创建 pod - named %s\n", pod.ObjectMeta.Name)
		if err := scheduler.Schedule(pod); err != nil {
			fmt.Println("调度出现异常:", err.Error())
		}
		time.Sleep(2 * time.Second)
	}
}

// watchK8sEvents 监听Kubernetes事件
func watchK8sEvents(scheduler *pkg.CustomScheduler) {
	watcher, err := scheduler.Clientset.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("创建Watch出错: %v\n", err)
		return
	}

	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}

		eventType := event.Type
		podName := pod.ObjectMeta.Name
		podNamespace := pod.ObjectMeta.Namespace

		fmt.Println("----> 监听到 Pod:", podName, "事件:", eventType, "<----")

		switch {
		case pod.Status.Phase == corev1.PodPending && eventType == watchapi.Added && pod.Spec.SchedulerName == scheduler.SchedulerName:
			// 非阻塞方式放入队列
			select {
			case podQueue <- pod:
				metrics.Lock()
				metrics.QueueLength = len(podQueue)
				metrics.Unlock()
				fmt.Printf("Pod %s 已加入调度队列\n", podName)
			default:
				fmt.Printf("警告: 调度队列已满, Pod %s 无法加入\n", podName)
			}

		case pod.Status.Phase == corev1.PodFailed && (pod.Status.Reason == "OutOfmemory" || pod.Status.Reason == "OutOfcpu"):
			fmt.Printf("检测到Pod %s/%s 资源不足, 将删除\n", podNamespace, podName)
			go deletePodWithRetry(scheduler, pod, 3, 2*time.Second)

		case eventType == watchapi.Deleted && pod.Spec.SchedulerName == scheduler.SchedulerName:
			scheduler.UpdateNodePods(pod)
		}
	}
}

// deletePodWithRetry 带重试机制的Pod删除
func deletePodWithRetry(scheduler *pkg.CustomScheduler, pod *corev1.Pod, maxRetries int, initialBackoff time.Duration) {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = scheduler.Clientset.CoreV1().Pods(pod.ObjectMeta.Namespace).Delete(context.TODO(), pod.ObjectMeta.Name, metav1.DeleteOptions{})
		if err == nil {
			fmt.Printf("成功删除Pod %s/%s\n", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
			return
		}

		backoff := initialBackoff * time.Duration(i+1)
		fmt.Printf("删除Pod %s/%s 失败 (尝试 %d/%d), %v. 将在 %v 后重试\n",
			pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, i+1, maxRetries, err, backoff)
		time.Sleep(backoff)
	}
	fmt.Printf("无法删除Pod %s/%s 最终错误: %v\n", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
}

// monitorClusterResources 监控集群资源
func monitorClusterResources() {
	ticker := time.NewTicker(monitorInterval)
	defer ticker.Stop()

	for range ticker.C {
		utils.MonitorAndWriteResources()
	}
}

// printMetrics 打印调度器指标
func printMetrics() {
	ticker := time.NewTicker(schedulerInterval)
	defer ticker.Stop()

	for range ticker.C {
		metrics.Lock()
		fmt.Printf("\n=== 调度器指标 ===\n")
		fmt.Printf("总调度Pod数: %d\n", metrics.TotalPodsScheduled)
		fmt.Printf("失败调度数: %d\n", metrics.FailedSchedules)
		fmt.Printf("当前活跃调度数: %d\n", metrics.ActiveSchedules)
		fmt.Printf("队列长度: %d\n", metrics.QueueLength)
		fmt.Printf("================\n\n")
		metrics.Unlock()
	}
}

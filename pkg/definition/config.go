package definition

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
)

// 常量定义
const (
	NAMESPACE      = "k8s"
	SchedulerName  = "custom-scheduler"
	MasterName     = "master"
	MasterIp       = "192.168.3.221"
	PrometheusPort = 31000
	JOB            = "node-exporter"
	CadvisorJob    = "cloud_cadvisor"
	SplittingChar  = "-"
	PodName        = ""
)

// 变量定义
var (
	ClientSet *kubernetes.Clientset

	CloudNodes   = []string{"master", "node1", "node2"}
	AmdEdgeNodes = []string{"edge1", "edge2", "edge3", "edge4"}
	ArmEdgeNodes = []string{"rasp1-arm", "rasp2-arm", "rasp3-arm", "rasp5-arm"}

	CloudNodesIps = map[string]string{
		"master": "192.168.3.221",
		"node1":  "192.168.3.222",
		"node2":  "192.168.3.223",
	}
	AmdEdgeNodesIps = map[string]string{
		"edge1": "192.168.3.225",
		"edge2": "192.168.3.216",
		"edge3": "192.168.3.217",
		"edge4": "192.168.3.226",
	}
	NodeIps = map[string]string{
		"master":    "192.168.3.221",
		"node1":     "192.168.3.222",
		"node2":     "192.168.3.223",
		"node3":     "192.168.3.212",
		"node4":     "192.168.3.224",
		"edge1":     "192.168.3.225",
		"edge2":     "192.168.3.216",
		"edge3":     "192.168.3.217",
		"edge4":     "192.168.3.226",
		"rasp1-arm": "192.168.3.205",
		"rasp2-arm": "192.168.3.209",
		"rasp3-arm": "192.168.3.207",
		"rasp4-arm": "192.168.3.201",
		"rasp5-arm": "192.168.3.220",
	}
	ArmEdgeNodesIps = map[string]string{
		"rasp1-arm": "192.168.3.205",
		"rasp2-arm": "192.168.3.209",
		"rasp3-arm": "192.168.3.207",
		"rasp5-arm": "192.168.3.220",
	}

	BasicOccupationCpu = map[string]float64{}
	BasicOccupationMem = map[string]float64{}

	// NodeCpuFreeURL 过去2分钟的CPU空闲率
	NodeCpuFreeURL = fmt.Sprintf(
		`avg by (instance)(rate(node_cpu_seconds_total{mode="idle",job="%s"}[2m]))`,
		JOB,
	)

	// NodeMemFreeURL 内存空闲率
	NodeMemFreeURL = fmt.Sprintf(
		`node_memory_MemAvailable_bytes{job="%s"} / node_memory_MemTotal_bytes{job="%s"}`,
		JOB, JOB,
	)

	// NodeCpuURL Node CPU使用量（毫核心）
	NodeCpuURL = fmt.Sprintf(
		`(1 - avg by (instance)(rate(node_cpu_seconds_total{mode="idle",job="%s"}[2m])))*`+
			`(count(count(node_cpu_seconds_total{job="%s"}) by (cpu,instance)) by (instance))*1000`,
		JOB, JOB,
	)

	// NodeMemURL Node内存使用量（字节）
	NodeMemURL = fmt.Sprintf(
		`node_memory_MemTotal_bytes{job="%s"} - node_memory_MemAvailable_bytes{job="%s"}`,
		JOB, JOB,
	)

	// PodCpuURL 示例：获取指定Pod的CPU使用量（需要传入podName变量）
	PodCpuURL = `sum(rate(container_cpu_usage_seconds_total{` +
		`container_label_io_kubernetes_container_name!="POD",job="%s",container_label_io_kubernetes_pod_name="%s"}[2m]))*1000`

	// PodMemURL 示例：获取指定Pod的内存使用量
	PodMemURL = `container_memory_usage_bytes{` +
		`container_label_io_kubernetes_container_name!="POD",job="%s",container_label_io_kubernetes_pod_name="%s"}`
)

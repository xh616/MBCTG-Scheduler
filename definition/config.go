package definition

import (
	"fmt"
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

	// NodeCpuFreeURL URL 构造（使用 fmt.Sprintf）
	NodeCpuFreeURL = fmt.Sprintf("http://%s:%d/api/v1/query?query=avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\", job=\"%s\"}[2m]))", MasterIp, PrometheusPort, JOB)
	NodeMemFreeURL = fmt.Sprintf("http://%s:%d/api/v1/query?query=node_memory_MemAvailable_bytes{job=\"%s\"}%%20/%%20node_memory_MemTotal_bytes{job=\"%s\"}", MasterIp, PrometheusPort, JOB, JOB)

	NodeCpuURL = fmt.Sprintf("http://%s:%d/api/v1/query?query=(1 - avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\", job=\"%s\"}[2m])))*(count(count(node_cpu_seconds_total{job=\"%s\"}) by (cpu, instance)) by (instance))*1000", MasterIp, PrometheusPort, JOB, JOB)
	NodeMemURL = fmt.Sprintf("http://%s:%d/api/v1/query?query=node_memory_MemTotal_bytes{job=\"%s\"} - node_memory_MemAvailable_bytes{job=\"%s\"}", MasterIp, PrometheusPort, JOB, JOB)

	PodCpuURL = fmt.Sprintf("http://%s:%d/api/v1/query?query=sum(rate(container_cpu_usage_seconds_total{container_label_io_kubernetes_container_name != \"POD\", job = \"%s\", container_label_io_kubernetes_pod_name=\"%s\"}[2m]))*1000", MasterIp, PrometheusPort, CadvisorJob, PodName)
	PodMemURL = fmt.Sprintf("http://%s:%d/api/v1/query?query=container_memory_usage_bytes{container_label_io_kubernetes_container_name != \"POD\", job=\"%s\", container_label_io_kubernetes_pod_name=\"%s\"}", MasterIp, PrometheusPort, CadvisorJob, PodName)
)

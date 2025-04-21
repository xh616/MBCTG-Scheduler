package utils

import (
	"MBCTG/definition"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// QueryResponse 定义 Prometheus 查询返回的 JSON 数据结构
type QueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []QueryResult `json:"result"`
	} `json:"data"`
}

// QueryResult 定义单个查询结果
type QueryResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"` // 第 0 个元素为时间戳，第 1 个为数值（字符串类型）
}

// HttpGetNodeMonitor 根据 req（"mem" 或 "cpu"）从 Prometheus 获取节点监控数据，并返回 map[node_name]value
func HttpGetNodeMonitor(req string) (map[string]float64, error) {
	var url string
	if req == "mem" {
		url = definition.NodeMemURL
	} else if req == "cpu" {
		url = definition.NodeCpuURL
	} else {
		return nil, errors.New("不支持的req类别")
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var qr QueryResponse
	if err := json.Unmarshal(body, &qr); err != nil {
		return nil, err
	}

	nodeMonitor := make(map[string]float64)
	// 处理返回数据，只有 status 为 "success" 时才处理
	if qr.Status == "success" {
		for _, v := range qr.Data.Result {
			// 从 metric 中获取 instance 字段
			nodeName := v.Metric["instance"]
			// 判断是否为云节点，如果不在 CLOUD_NODES 列表中，则调用辅助函数根据 IP 获取节点名称
			if !contains(definition.CloudNodes, nodeName) {
				// 替换 ":9100" 后再调用
				instanceIP := strings.ReplaceAll(nodeName, ":9100", "")
				nodeName = GetNodeNameByIP(instanceIP)
			}
			// 获取数值，v.Value[1] 通常为字符串类型
			if len(v.Value) < 2 {
				continue
			}
			valueStr, ok := v.Value[1].(string)
			if !ok {
				continue
			}
			nodeVal, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				continue
			}
			nodeMonitor[nodeName] = nodeVal
		}
	}
	return nodeMonitor, nil
}

// HttpGetNodeFreeRateMonitor 根据 req（"mem" 或 "cpu"）获取节点空闲率监控数据
func HttpGetNodeFreeRateMonitor(req string) (map[string]float64, error) {
	var url string
	if req == "mem" {
		url = definition.NodeMemFreeURL
	} else if req == "cpu" {
		url = definition.NodeCpuFreeURL
	} else {
		return nil, errors.New("不支持的请求类型")
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var qr QueryResponse
	if err := json.Unmarshal(body, &qr); err != nil {
		return nil, err
	}

	nodeMonitor := make(map[string]float64)
	if qr.Status == "success" {
		for _, v := range qr.Data.Result {
			nodeName := v.Metric["instance"]
			if !contains(definition.CloudNodes, nodeName) {
				instanceIP := strings.ReplaceAll(nodeName, ":9100", "")
				nodeName = GetNodeNameByIP(instanceIP)
			}
			if len(v.Value) < 2 {
				continue
			}
			valueStr, ok := v.Value[1].(string)
			if !ok {
				continue
			}
			nodeVal, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				continue
			}
			nodeMonitor[nodeName] = nodeVal
		}
	}
	return nodeMonitor, nil
}

// HttpGetPodMonitor 根据 req（"mem" 或 "cpu"）和 podName 获取 Pod 监控数据
func HttpGetPodMonitor(req, podName string) (float64, error) {
	var val float64 = 0
	var url string

	if req == "mem" {
		// 将 URL 中的 {POD_NAME} 替换为实际的 podName
		url = strings.ReplaceAll(definition.PodMemURL, "{POD_NAME}", podName)
		resp, err := http.Get(url)
		if err != nil {
			return 0, err
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {

			}
		}(resp.Body)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}
		var qr QueryResponse
		if err := json.Unmarshal(body, &qr); err != nil {
			return 0, err
		}
		if qr.Status == "success" {
			// 对于内存监控，取所有返回值中的最大值
			for _, v := range qr.Data.Result {
				if len(v.Value) < 2 {
					continue
				}
				valueStr, ok := v.Value[1].(string)
				if !ok {
					continue
				}
				valTemp, err := strconv.ParseFloat(valueStr, 64)
				if err != nil {
					continue
				}
				if valTemp > val {
					val = valTemp
				}
			}
		}
	} else if req == "cpu" {
		url = strings.ReplaceAll(definition.PodCpuURL, "{POD_NAME}", podName)
		resp, err := http.Get(url)
		if err != nil {
			return 0, err
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {

			}
		}(resp.Body)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}
		var qr QueryResponse
		if err := json.Unmarshal(body, &qr); err != nil {
			return 0, err
		}
		if qr.Status == "success" && len(qr.Data.Result) > 0 {
			v := qr.Data.Result[0]
			if len(v.Value) >= 2 {
				valueStr, ok := v.Value[1].(string)
				if ok {
					valTemp, err := strconv.ParseFloat(valueStr, 64)
					if err == nil {
						val += valTemp
					}
				}
			}
		}
	} else {
		return 0, errors.New("unsupported req type")
	}
	return val, nil
}

func main() {
	// 示例调用
	nodeMonitorCPU, err := HttpGetNodeMonitor("cpu")
	if err != nil {
		fmt.Println("HttpGetNodeMonitor(cpu) error:", err)
	} else {
		fmt.Println("Node CPU Monitor:", nodeMonitorCPU)
	}

	nodeMonitorMem, err := HttpGetNodeMonitor("mem")
	if err != nil {
		fmt.Println("HttpGetNodeMonitor(mem) error:", err)
	} else {
		fmt.Println("Node Mem Monitor:", nodeMonitorMem)
	}

	podMem, err := HttpGetPodMonitor("mem", "pod-1")
	if err != nil {
		fmt.Println("HttpGetPodMonitor(mem, pod-1) error:", err)
	} else {
		// 转换为 MB
		fmt.Printf("Pod pod-1 Memory: %.2f MB\n", podMem/1024/1024)
	}

	podCpu, err := HttpGetPodMonitor("cpu", "pod-1")
	if err != nil {
		fmt.Println("HttpGetPodMonitor(cpu, pod-1) error:", err)
	} else {
		// 转换为 CPU 核（假设返回值单位为毫核）
		fmt.Printf("Pod pod-1 CPU: %.2f cores\n", podCpu/1000)
	}
}

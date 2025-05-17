package utils

import (
	"MBCTG/pkg/definition"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// MetricResult matches individual Prometheus metric entries.
type MetricResult struct {
	Metric struct {
		Instance string `json:"instance"`
	} `json:"metric"`
	Value []interface{} `json:"value"` // [<timestamp>, "<value-as-string>"]
}

type queryResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []MetricResult `json:"result"`
	} `json:"data"`
}

// performQuery 处理promQL
func performQuery(promql string) ([]MetricResult, error) {
	endpoint := fmt.Sprintf("http://%s:%d/api/v1/query", definition.MasterIp, definition.PrometheusPort)
	params := url.Values{}
	params.Set("query", promql)
	fullURL := endpoint + "?" + params.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var qr queryResponse
	if err := json.Unmarshal(body, &qr); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	if qr.Status != "success" {
		return nil, fmt.Errorf("prometheus status: %s", qr.Status)
	}
	return qr.Data.Result, nil
}

// parseResultsToMap Node监控数据转为map
func parseResultsToMap(results []MetricResult) (map[string]float64, error) {
	nodeMonitor := make(map[string]float64)
	for _, item := range results {
		name := item.Metric.Instance
		if !Contains(definition.CloudNodes, name) {
			continue
		}
		raw, ok := item.Value[1].(string)
		if !ok {
			return nil, fmt.Errorf("unexpected value type for %s: %T", name, item.Value[1])
		}
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value for %s: %v", name, err)
		}
		nodeMonitor[name] = val
	}
	return nodeMonitor, nil
}

// HttpGetNodeMonitor 监控节点cpu和内存使用量
func HttpGetNodeMonitor(req string) (map[string]float64, error) {
	var promql string
	switch req {
	case "mem":
		promql = definition.NodeMemURL
	case "cpu":
		promql = definition.NodeCpuURL
	default:
		return nil, errors.New("unsupported request type")
	}
	results, err := performQuery(promql)
	if err != nil {
		return nil, err
	}
	return parseResultsToMap(results)
}

// HttpGetNodeFreeRateMonitor 监控节点cpu和内存空闲率
func HttpGetNodeFreeRateMonitor(req string) (map[string]float64, error) {
	var promql string
	switch req {
	case "mem":
		promql = definition.NodeMemFreeURL
	case "cpu":
		promql = definition.NodeCpuFreeURL
	default:
		return nil, errors.New("unsupported request type")
	}
	results, err := performQuery(promql)
	if err != nil {
		return nil, err
	}
	return parseResultsToMap(results)
}

// HttpGetPodMonitor 监控pod的cpu和内存使用量
func HttpGetPodMonitor(req, podName string) (float64, error) {
	var promql string
	switch req {
	case "mem":
		promql = fmt.Sprintf(definition.PodMemURL, definition.CadvisorJob, podName)
	case "cpu":
		promql = fmt.Sprintf(definition.PodCpuURL, definition.CadvisorJob, podName)
	default:
		return 0, errors.New("unsupported request type")
	}
	results, err := performQuery(promql)
	if err != nil {
		return 0, err
	}
	var total float64
	for _, item := range results {
		raw, ok := item.Value[1].(string)
		if !ok {
			return 0, fmt.Errorf("unexpected value type: %T", item.Value[1])
		}
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse value: %v", err)
		}
		total += val
	}
	return total, nil
}

func PrintNodeMonitorToRead(req string) error {
	monitor, _ := HttpGetNodeMonitor(req)
	var divisor float64
	var unit string
	switch req {
	case "mem":
		divisor = 1 << 30
		unit = "GB"
	case "cpu":
		divisor = 1000
		unit = "核"
	default:
		return errors.New("unsupported request type")
	}
	for key, val := range monitor {
		monitor[key] = val / divisor
	}
	fmt.Println(req, "(", unit, "):", monitor)
	return nil
}

func PrintPodMonitorToRead(req, podName string) error {
	val, _ := HttpGetPodMonitor(req, podName)
	var divisor float64
	var unit string
	switch req {
	case "mem":
		divisor = 1 << 20
		unit = "MB"
	case "cpu":
		divisor = 1000
		unit = "核"
	default:
		return errors.New("unsupported request type")
	}
	fmt.Printf("%s %s:%f(%s)\n", podName, req, val/divisor, unit)
	return nil
}

// MonitorAndWriteResources 监控并写入资源数据
func MonitorAndWriteResources() {
	nodesCPU, err := HttpGetNodeMonitor("cpu")
	if err != nil {
		fmt.Printf("获取CPU数据错误: %v\n", err)
		return
	}

	nodesMem, err := HttpGetNodeMonitor("mem")
	if err != nil {
		fmt.Printf("获取Mem数据错误: %v\n", err)
		return
	}

	cpuUsage := make(map[string]float64)
	memUsage := make(map[string]float64)
	for key, value := range nodesCPU {
		cpuUsage[key] = value / 1000.0
	}
	for key, value := range nodesMem {
		memUsage[key] = value / (1 << 30)
	}

	currentTime := time.Now().Format(time.RFC3339)
	content := fmt.Sprintf("time: %s\nCPU: %v\nMem: %v\n", currentTime, cpuUsage, memUsage)

	if err := writeToFile("node_resource.txt", content); err != nil {
		fmt.Printf("写入文件错误: %v\n", err)
	}
}

// writeToFile 写入文件
func writeToFile(filename, content string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(f)

	if _, err := f.WriteString(content); err != nil {
		return err
	}
	return nil
}

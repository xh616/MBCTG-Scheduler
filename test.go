package main

import (
	"MBCTG/pkg/definition"
	"fmt"
)

func main() {
	promUrl := definition.PodCpuURL
	podName := "demo1"
	promUrl = fmt.Sprintf(promUrl, definition.CadvisorJob, podName)
	fmt.Println(promUrl)
}

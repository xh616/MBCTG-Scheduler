## 基于合作博弈论的k8s负载均衡调度器

### 修改kube\config
即集群中的`$HOME/.kube/config` 文件

### 部署Prometheus + Node Exporter + cAdvisor

### 修改config.go
修改自己集群的相关配置

### 直接运行或打包镜像部署均可

### pod yaml指定示例

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: demo1
  namespace: k8s
spec:
  schedulerName: custom-scheduler
  containers:
    - name: demo1
      image: polinux/stress
      resources:
        limits:
          cpu: "1"
          memory: "888Mi"
        requests:
          cpu: "0.8"
          memory: "502Mi"
      command: ["stress"]
      args: ["--vm", "1", "--vm-bytes", "500M", "--vm-hang", "10000", "--cpu", "1"]

```
# ClusterRole 压力测试工具

## 功能说明

这个工具用于向Kubernetes API Server发送大量创建ClusterRole的请求，用于压力测试。

**特点：**
- 可指定总请求次数
- 可配置并发数
- 每个ClusterRole的名字大小约为 **500KB**
- 自动清理创建的资源
- 实时显示成功/失败统计

## 使用方法

### 1. 准备kubeconfig文件

将你的kubeconfig文件放在 `../client/kubeconfig` 路径，或使用 `-kubeconfig` 参数指定路径。

### 2. 安装依赖

```bash
cd etcd
go mod download
```

### 3. 运行测试

```bash
# 基本用法（默认：100次请求，10并发）
go run main.go

# 自定义参数
go run main.go -n 500 -c 20

# 指定kubeconfig路径
go run main.go -kubeconfig /path/to/kubeconfig -n 1000 -c 50
```

### 4. 编译后运行

```bash
go build -o clusterrole-test.exe
./clusterrole-test.exe -n 500 -c 20
```

## 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-n` | 总请求次数 (Total) | 100 |
| `-c` | 并发数 (Concurrency) | 10 |
| `-kubeconfig` | kubeconfig文件路径 | `../client/kubeconfig` |

## 示例输出

```
开始测试: 总请求数 100, 并发数 10
每个ClusterRole名字大小: 约500KB

✅ [0] 成功创建 (耗时: 250ms, 名字长度: 512000 字节)
✅ [10] 成功创建 (耗时: 230ms, 名字长度: 512000 字节)
✅ [20] 成功创建 (耗时: 245ms, 名字长度: 512000 字节)
...

============================================================
测试完成!
总耗时: 25.5s
成功: 98
失败: 2
总请求: 100
平均耗时: 260ms
QPS: 3.84
============================================================
```

## 注意事项

⚠️ **重要提醒：**

1. **名字大小限制**：Kubernetes通常对资源名称有长度限制（一般为253字符），这个工具生成500KB的名字可能会被API Server拒绝。这是一个压力测试工具，主要用于测试API Server对超大名字的处理能力。

2. **资源清理**：程序会在创建成功后自动删除ClusterRole，但如果程序异常退出，可能需要手动清理：
   ```bash
   kubectl get clusterroles | grep "clusterrole-" | awk '{print $1}' | xargs kubectl delete clusterrole
   ```

3. **API Server压力**：高并发和大量请求可能对API Server造成较大压力，请在测试环境中谨慎使用。

4. **权限要求**：需要有创建和删除ClusterRole的权限。

## 故障排查

### 错误：无法连接到集群

确保kubeconfig文件路径正确且内容有效。

### 错误：权限不足

确保kubeconfig中的用户有足够权限操作ClusterRole资源：
```bash
kubectl auth can-i create clusterroles
kubectl auth can-i delete clusterroles
```

### 所有请求都失败

检查API Server是否正常运行，以及是否有名称长度限制：
```bash
kubectl cluster-info
```

# 算法扩展指南

本系统采用了高度解耦的动态注册中心架构。你可以无需修改现有业务路由，直接插入任意新的 AI/NLP 算法。

## 1. 确定你的算法属于哪个阶段
系统分为 4 个标准接口（位于 `pkg/algorithm/interface.go`）：
- `FilterAlgorithm`: 用于过滤无用数据（返回 true/false）。
- `RewriteAlgorithm`: 接收一段文本，输出改写后的文本。
- `DistillAlgorithm`: 用于蒸馏提取（如 Logits, 特征提取）。
- `SyntheticAlgorithm`: 根据提示词生成新的复杂数据（如 DPO, Few-shot）。

## 2. 编写算法实现类
以“添加一个将文本转为大写的重写算法”为例。
在 `pkg/rewrite/uppercase.go` 中新建文件：

```go
package rewrite

import (
	"data-flywheel/internal/external"
	"strings"
)

type UppercaseRewrite struct{}

// 必须实现接口规定的 Name() 方法，这是 API 调用时使用的名字
func (r *UppercaseRewrite) Name() string { return "uppercase" }

// 必须实现 Rewrite 方法
func (r *UppercaseRewrite) Rewrite(text string, params map[string]interface{}, vllm *external.VLLMClient) (string, error) {
    // 你可以从 params 中读取动态参数
    // 你可以直接调用 vllm 获取大模型能力
	return strings.ToUpper(text), nil
}
```

## 3. 在 main.go 中注册算法
打开 `main.go`，在 `main()` 函数顶部的注册区域加一行代码：
```go
service.RegisterRewrite(&rewrite.UppercaseRewrite{})
```

## 4. 通过 API 动态调用
启动程序后，你可以立刻通过统一动态路由调用新算法：
```bash
curl -X POST http://localhost:8080/api/dynamic/rewrite \
-H "Content-Type: application/json" \
-d '{
    "algorithm": "uppercase",
    "text": "hello world"
}'
```
package algorithm

// FilterAlgorithm 定义了文本过滤算法的标准接口
type FilterAlgorithm interface {
	// Name 算法的唯一标识
	Name() string
	// Evaluate 评估文本，返回 true 表示保留，false 表示丢弃，并附带原因
	Evaluate(text string, params map[string]interface{}) (keep bool, reason string)
}
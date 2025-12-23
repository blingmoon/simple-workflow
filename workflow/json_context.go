package workflow

import (
	"encoding/json"
	"fmt"
)

// JSONContext 封装 JSON 上下文，提供便捷的读写方法
type JSONContext struct {
	data map[string]any
}

// NewJSONContext 从字节创建 JSON 上下文
func NewJSONContext(b []byte) *JSONContext {
	ctx := &JSONContext{
		data: make(map[string]any),
	}
	if len(b) > 0 {
		json.Unmarshal(b, &ctx.data)
	}
	return ctx
}

// NewJSONContextFromMap 从 map 创建上下文
func NewJSONContextFromMap(m map[string]any) *JSONContext {
	if m == nil {
		m = make(map[string]any)
	}
	return &JSONContext{data: m}
}

// Get 获取值，支持嵌套路径
// 例如: Get("user", "name") 获取 user.name
func (c *JSONContext) Get(keys ...string) (any, bool) {
	if len(keys) == 0 {
		return nil, false
	}

	current := any(c.data)
	for _, key := range keys {
		if currentMap, ok := current.(map[string]any); ok {
			if val, exists := currentMap[key]; exists {
				current = val
			} else {
				return nil, false
			}
		} else {
			return nil, false
		}
	}
	return current, true
}

// GetString 获取字符串值
func (c *JSONContext) GetString(keys ...string) (string, bool) {
	val, ok := c.Get(keys...)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetInt64 获取 int64 值
func (c *JSONContext) GetInt64(keys ...string) (int64, bool) {
	val, ok := c.Get(keys...)
	if !ok {
		return 0, false
	}

	// 尝试多种数字类型
	switch v := val.(type) {
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	default:
		return 0, false
	}
}

// GetFloat64 获取 float64 值
func (c *JSONContext) GetFloat64(keys ...string) (float64, bool) {
	val, ok := c.Get(keys...)
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}

// GetBool 获取布尔值
func (c *JSONContext) GetBool(keys ...string) (bool, bool) {
	val, ok := c.Get(keys...)
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

// Set 设置值，支持嵌套路径
// 例如: Set([]string{"user", "name"}, "张三") 设置 user.name = "张三"
func (c *JSONContext) Set(keys []string, value any) error {
	if len(keys) == 0 {
		return fmt.Errorf("keys cannot be empty")
	}

	// 确保所有中间路径都是 map
	current := c.data
	for i := 0; i < len(keys)-1; i++ {
		key := keys[i]
		if _, ok := current[key]; !ok {
			current[key] = make(map[string]any)
		}

		nextMap, ok := current[key].(map[string]any)
		if !ok {
			// 如果不是 map，覆盖它
			nextMap = make(map[string]any)
			current[key] = nextMap
		}
		current = nextMap
	}

	// 设置最终值
	current[keys[len(keys)-1]] = value
	return nil
}

// Delete 删除指定路径的值
func (c *JSONContext) Delete(keys ...string) {
	if len(keys) == 0 {
		return
	}

	if len(keys) == 1 {
		delete(c.data, keys[0])
		return
	}

	// 找到父级 map
	current := c.data
	for i := 0; i < len(keys)-1; i++ {
		if nextMap, ok := current[keys[i]].(map[string]any); ok {
			current = nextMap
		} else {
			return
		}
	}
	delete(current, keys[len(keys)-1])
}

// ToBytes 转换为 JSON 字节
func (c *JSONContext) ToBytes() ([]byte, error) {
	return json.Marshal(c.data)
}
func (c *JSONContext) ToBytesWithoutError() []byte {
	bytes, err := json.Marshal(c.data)
	if err != nil {
		return nil
	}
	return bytes
}

// ToRawMessage 转换为 json.RawMessage
func (c *JSONContext) ToRawMessage() (json.RawMessage, error) {
	b, err := c.ToBytes()
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// ToMap 返回底层 map（注意：返回的是引用）
func (c *JSONContext) ToMap() map[string]any {
	return c.data
}

// Clone 深拷贝上下文
func (c *JSONContext) Clone() *JSONContext {
	b, _ := c.ToBytes()
	return NewJSONContext(b)
}

// Unmarshal 将上下文反序列化到指定结构体
func (c *JSONContext) Unmarshal(v any) error {
	b, err := c.ToBytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// MergeJSONContexts 合并多个上下文（后面的会覆盖前面的）
func MergeJSONContexts(contexts ...*JSONContext) *JSONContext {
	result := NewJSONContext(nil)
	for _, ctx := range contexts {
		if ctx != nil {
			for k, v := range ctx.data {
				result.data[k] = v
			}
		}
	}
	return result
}

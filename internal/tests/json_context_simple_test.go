package tests

import (
	"testing"

	"github.com/blingmoon/simple-workflow/workflow"
	"github.com/stretchr/testify/assert"
)

// TestJSONContextSimple 测试 JSON 上下文的基本功能
func TestJSONContextSimple(t *testing.T) {
	t.Run("创建和读取", func(t *testing.T) {
		// 从 JSON 创建
		jsonStr := `{"name":"test","age":18}`
		ctx := workflow.NewJSONContext([]byte(jsonStr))

		name, ok := ctx.GetString("name")
		assert.True(t, ok)
		assert.Equal(t, "test", name)

		age, ok := ctx.GetInt64("age")
		assert.True(t, ok)
		assert.Equal(t, int64(18), age)
	})

	t.Run("从 map 创建", func(t *testing.T) {
		data := map[string]any{
			"user":  "alice",
			"score": 95.5,
		}

		ctx := workflow.NewJSONContextFromMap(data)
		user, ok := ctx.GetString("user")
		assert.True(t, ok)
		assert.Equal(t, "alice", user)
	})

	t.Run("设置和获取值", func(t *testing.T) {
		ctx := workflow.NewJSONContext(nil)

		// 设置值
		err := ctx.Set([]string{"key1"}, "value1")
		assert.NoError(t, err)

		err = ctx.Set([]string{"key2"}, 123)
		assert.NoError(t, err)

		// 获取值
		val1, ok := ctx.GetString("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", val1)

		val2, ok := ctx.GetInt64("key2")
		assert.True(t, ok)
		assert.Equal(t, int64(123), val2)
	})

	t.Run("嵌套路径", func(t *testing.T) {
		ctx := workflow.NewJSONContext(nil)

		// 设置嵌套值
		err := ctx.Set([]string{"user", "name"}, "张三")
		assert.NoError(t, err)

		err = ctx.Set([]string{"user", "age"}, 30)
		assert.NoError(t, err)

		// 获取嵌套值
		name, ok := ctx.GetString("user", "name")
		assert.True(t, ok)
		assert.Equal(t, "张三", name)

		age, ok := ctx.GetInt64("user", "age")
		assert.True(t, ok)
		assert.Equal(t, int64(30), age)
	})

	t.Run("转换为字节", func(t *testing.T) {
		ctx := workflow.NewJSONContextFromMap(map[string]any{
			"key": "value",
		})

		bytes, err := ctx.ToBytes()
		assert.NoError(t, err)
		assert.NotEmpty(t, bytes)

		// 从字节恢复
		ctx2 := workflow.NewJSONContext(bytes)
		val, ok := ctx2.GetString("key")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("克隆", func(t *testing.T) {
		original := workflow.NewJSONContextFromMap(map[string]any{
			"name": "original",
		})

		cloned := original.Clone()
		_ = cloned.Set([]string{"name"}, "cloned")

		// 验证原始对象未改变
		name, _ := original.GetString("name")
		assert.Equal(t, "original", name)

		clonedName, _ := cloned.GetString("name")
		assert.Equal(t, "cloned", clonedName)
	})
}


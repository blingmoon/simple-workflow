package workflow

import (
	"encoding/json"
	"testing"
)

func TestJSONContext_BasicOperations(t *testing.T) {
	// 创建空上下文
	ctx := NewJSONContext(nil)

	// 设置值
	ctx.Set([]string{"user", "name"}, "张三")
	ctx.Set([]string{"user", "age"}, int64(25))
	ctx.Set([]string{"user", "active"}, true)
	ctx.Set([]string{"score"}, 98.5)

	// 获取值
	name, ok := ctx.GetString("user", "name")
	if !ok || name != "张三" {
		t.Errorf("Expected name=张三, got %s", name)
	}

	age, ok := ctx.GetInt64("user", "age")
	if !ok || age != 25 {
		t.Errorf("Expected age=25, got %d", age)
	}

	active, ok := ctx.GetBool("user", "active")
	if !ok || !active {
		t.Errorf("Expected active=true, got %v", active)
	}

	score, ok := ctx.GetFloat64("score")
	if !ok || score != 98.5 {
		t.Errorf("Expected score=98.5, got %f", score)
	}
}

func TestJSONContext_FromBytes(t *testing.T) {
	// 从 JSON 字节创建
	jsonData := []byte(`{
		"workflow_id": 12345,
		"task_type": "审核",
		"node_event": {
			"event_content": "审核通过",
			"event_ts": 1640000000
		}
	}`)

	ctx := NewJSONContext(jsonData)

	// 读取嵌套值
	workflowID, ok := ctx.GetInt64("workflow_id")
	if !ok || workflowID != 12345 {
		t.Errorf("Expected workflow_id=12345, got %d", workflowID)
	}

	eventContent, ok := ctx.GetString("node_event", "event_content")
	if !ok || eventContent != "审核通过" {
		t.Errorf("Expected event_content=审核通过, got %s", eventContent)
	}

	eventTs, ok := ctx.GetInt64("node_event", "event_ts")
	if !ok || eventTs != 1640000000 {
		t.Errorf("Expected event_ts=1640000000, got %d", eventTs)
	}
}

func TestJSONContext_ToBytes(t *testing.T) {
	ctx := NewJSONContext(nil)
	ctx.Set([]string{"name"}, "测试")
	ctx.Set([]string{"count"}, int64(100))

	// 转换为字节
	b, err := ctx.ToBytes()
	if err != nil {
		t.Fatalf("ToBytes failed: %v", err)
	}

	// 验证 JSON
	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if result["name"] != "测试" {
		t.Errorf("Expected name=测试, got %v", result["name"])
	}
}

func TestJSONContext_Delete(t *testing.T) {
	ctx := NewJSONContext([]byte(`{
		"field1": "value1",
		"nested": {
			"field2": "value2"
		}
	}`))

	// 删除顶层字段
	ctx.Delete("field1")
	_, ok := ctx.GetString("field1")
	if ok {
		t.Error("field1 should be deleted")
	}

	// 删除嵌套字段
	ctx.Delete("nested", "field2")
	_, ok = ctx.GetString("nested", "field2")
	if ok {
		t.Error("nested.field2 should be deleted")
	}
}

func TestJSONContext_Clone(t *testing.T) {
	original := NewJSONContext([]byte(`{"name": "原始"}`))
	cloned := original.Clone()

	// 修改克隆
	cloned.Set([]string{"name"}, "克隆")

	// 验证原始未被修改
	name, _ := original.GetString("name")
	if name != "原始" {
		t.Errorf("Original should not be modified, got %s", name)
	}

	clonedName, _ := cloned.GetString("name")
	if clonedName != "克隆" {
		t.Errorf("Cloned should be modified, got %s", clonedName)
	}
}

func TestJSONContext_Unmarshal(t *testing.T) {
	ctx := NewJSONContext([]byte(`{
		"user_id": "123",
		"age": 25,
		"email": "test@example.com"
	}`))

	// 反序列化到结构体
	type User struct {
		UserID string `json:"user_id"`
		Age    int    `json:"age"`
		Email  string `json:"email"`
	}

	var user User
	if err := ctx.Unmarshal(&user); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if user.UserID != "123" || user.Age != 25 || user.Email != "test@example.com" {
		t.Errorf("Unmarshal result incorrect: %+v", user)
	}
}

func TestMergeJSONContexts(t *testing.T) {
	ctx1 := NewJSONContext([]byte(`{"a": 1, "b": 2}`))
	ctx2 := NewJSONContext([]byte(`{"b": 3, "c": 4}`))

	merged := MergeJSONContexts(ctx1, ctx2)

	// b 应该被 ctx2 覆盖
	b, _ := merged.GetInt64("b")
	if b != 3 {
		t.Errorf("Expected b=3, got %d", b)
	}

	// a 和 c 都应该存在
	a, ok1 := merged.GetInt64("a")
	c, ok2 := merged.GetInt64("c")
	if !ok1 || a != 1 || !ok2 || c != 4 {
		t.Error("Merge failed")
	}
}

// 性能测试
func BenchmarkJSONContext_Get(b *testing.B) {
	ctx := NewJSONContext([]byte(`{
		"level1": {
			"level2": {
				"level3": {
					"value": "test"
				}
			}
		}
	}`))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.GetString("level1", "level2", "level3", "value")
	}
}

func BenchmarkJSONContext_Set(b *testing.B) {
	ctx := NewJSONContext(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Set([]string{"level1", "level2", "value"}, "test")
	}
}

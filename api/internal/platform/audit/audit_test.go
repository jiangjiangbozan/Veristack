package audit

import (
	"encoding/json"
	"testing"
)

func TestMarshalRedactedReplacesSensitiveFields(t *testing.T) {
	cases := []struct {
		name    string
		payload any
		want    map[string]any
	}{
		{
			name:    "正常值：令牌字段被替换为占位符",
			payload: map[string]any{"token": "abc123", "expiresAt": "2026-01-01T00:00:00+08:00"},
			want:    map[string]any{"token": redactedPlaceholder, "expiresAt": "2026-01-01T00:00:00+08:00"},
		},
		{
			name:    "边界值：空字符串令牌同样被替换",
			payload: map[string]any{"token": ""},
			want:    map[string]any{"token": redactedPlaceholder},
		},
		{
			name:    "非法输入：非字符串的令牌字段同样被替换",
			payload: map[string]any{"accessToken": 12345},
			want:    map[string]any{"accessToken": redactedPlaceholder},
		},
		{
			name:    "异常场景：嵌套结构中的敏感字段同样被替换",
			payload: map[string]any{"outer": map[string]any{"password": "p@ss", "name": "keep"}},
			want:    map[string]any{"outer": map[string]any{"password": redactedPlaceholder, "name": "keep"}},
		},
		{
			name:    "异常场景：数组元素中的敏感字段同样被替换",
			payload: map[string]any{"items": []any{map[string]any{"secret": "s"}}, "count": 1},
			want:    map[string]any{"items": []any{map[string]any{"secret": redactedPlaceholder}}, "count": float64(1)},
		},
		{
			name:    "正常值：非敏感字段保持原样",
			payload: map[string]any{"title": "首页主题", "version": 2},
			want:    map[string]any{"title": "首页主题", "version": float64(2)},
		},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			raw, err := marshalRedacted(item.payload)
			if err != nil {
				t.Fatalf("序列化失败: %v", err)
			}
			var got map[string]any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("反序列化失败: %v", err)
			}
			assertSame(t, item.want, got)
		})
	}
}

func TestMarshalRedactedHandlesNilPayload(t *testing.T) {
	raw, err := marshalRedacted(nil)
	if err != nil {
		t.Fatalf("序列化空载荷不应报错: %v", err)
	}
	if raw != nil {
		t.Fatalf("空载荷应当返回 nil，实际为 %s", string(raw))
	}
}

func TestIsSensitiveKey(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{key: "token", want: true},
		{key: "Token", want: true},
		{key: "accessToken", want: true},
		{key: "passwordHash", want: true},
		{key: "apiKey", want: true},
		{key: "privateKey", want: true},
		{key: "title", want: false},
		{key: "expiresAt", want: false},
		{key: "tokenIssued", want: true},
	}
	for _, item := range cases {
		t.Run(item.key, func(t *testing.T) {
			if got := isSensitiveKey(item.key); got != item.want {
				t.Fatalf("isSensitiveKey(%q) = %v，期望 %v", item.key, got, item.want)
			}
		})
	}
}

// assertSame 逐字段比较两个 JSON 对象，失败时给出可读差异。
func assertSame(t *testing.T, want, got map[string]any) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("字段数量不一致，期望 %d，实际 %d", len(want), len(got))
	}
	for key, expected := range want {
		actual, ok := got[key]
		if !ok {
			t.Fatalf("缺少字段 %q", key)
		}
		assertSameValue(t, key, expected, actual)
	}
}

// assertSameValue 比较单个字段值，嵌套对象与数组逐层展开。
func assertSameValue(t *testing.T, path string, want, got any) {
	t.Helper()
	wantMap, wantIsMap := want.(map[string]any)
	gotMap, gotIsMap := got.(map[string]any)
	if wantIsMap && gotIsMap {
		assertSame(t, wantMap, gotMap)
		return
	}
	wantList, wantIsList := want.([]any)
	gotList, gotIsList := got.([]any)
	if wantIsList && gotIsList {
		if len(wantList) != len(gotList) {
			t.Fatalf("字段 %q 的数组长度不一致，期望 %d，实际 %d", path, len(wantList), len(gotList))
		}
		for index := range wantList {
			assertSameValue(t, path, wantList[index], gotList[index])
		}
		return
	}
	if want != got {
		t.Fatalf("字段 %q 不一致，期望 %v，实际 %v", path, want, got)
	}
}

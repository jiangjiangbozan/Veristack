package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"api/internal/platform/config"
)

const (
	// defaultModel 是未显式指定时使用的模型标识。
	defaultModel = "gpt-4o-mini"
	// requestTimeout 限制单次模型调用耗时，避免阻塞业务请求。
	requestTimeout = 60 * time.Second
	// maxDocumentChars 限制送入模型的文档长度，控制成本与延迟。
	maxDocumentChars = 24000
)

// OpenAICompatible 通过 OpenAI 兼容的 Chat Completions 接口调用模型。
type OpenAICompatible struct {
	APIKey  string       // 接口密钥
	BaseURL string       // 接口地址
	Client  *http.Client // HTTP 客户端
	Model   string       // 模型标识
}

// NewOpenAICompatible 依据配置创建模型适配器，未配置密钥时返回不可用实例。
func NewOpenAICompatible(cfg config.AIConfig) *OpenAICompatible {
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	return &OpenAICompatible{
		APIKey:  cfg.APIKey,
		BaseURL: strings.TrimRight(cfg.BaseURL, "/"),
		Client:  &http.Client{Timeout: requestTimeout},
		Model:   model,
	}
}

// AnalyzeChangeImpact 分析变更对现有需求的影响。
func (m *OpenAICompatible) AnalyzeChangeImpact(ctx context.Context, baselineText, changeText string) (Result, error) {
	prompt := fmt.Sprintf("请分析下列需求变更对既有需求范围的影响，输出受影响的需求编号与影响说明。\n既有需求：\n%s\n变更内容：\n%s", truncate(baselineText), truncate(changeText))
	return m.complete(ctx, prompt)
}

// ExtractRequirements 从需求文档中提取候选需求。
func (m *OpenAICompatible) ExtractRequirements(ctx context.Context, document string) (Result, error) {
	prompt := fmt.Sprintf("请从下列原始需求文档中提取候选需求条目，输出编号、标题、描述、优先级与验收条件。\n%s", truncate(document))
	return m.complete(ctx, prompt)
}

// Name 返回模型标识。
func (m *OpenAICompatible) Name() string {
	return m.Model
}

// SummarizeDelivery 为交付版本生成自然语言摘要。
func (m *OpenAICompatible) SummarizeDelivery(ctx context.Context, deliveryText string) (Result, error) {
	prompt := fmt.Sprintf("请为下列软件交付内容生成一段面向客户理解的自然语言摘要，说明交付了什么、依据什么验收。\n%s", truncate(deliveryText))
	return m.complete(ctx, prompt)
}

// complete 调用模型接口并解析结构化输出。
func (m *OpenAICompatible) complete(ctx context.Context, prompt string) (Result, error) {
	if m.APIKey == "" {
		return Result{}, fmt.Errorf("未配置模型接口密钥")
	}
	body := map[string]any{
		"model": m.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个软件交付分析助手，只输出 JSON，不做额外解释。"},
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Result{}, fmt.Errorf("序列化模型请求失败: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, m.BaseURL+"/chat/completions", strings.NewReader(string(payload)))
	if err != nil {
		return Result{}, fmt.Errorf("构造模型请求失败: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+m.APIKey)
	response, err := m.Client.Do(request)
	if err != nil {
		return Result{}, fmt.Errorf("调用模型失败: %w", err)
	}
	defer response.Body.Close()
	var decoded chatResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return Result{}, fmt.Errorf("解析模型响应失败: %w", err)
	}
	return parseModelContent(decoded)
}

// chatResponse 是模型接口的响应结构。
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// parseModelContent 从模型输出中解析出结构化建议。
func parseModelContent(response chatResponse) (Result, error) {
	if len(response.Choices) == 0 {
		return Result{}, fmt.Errorf("模型未返回任何结果")
	}
	content := strings.TrimSpace(response.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(content, "```")
	var result Result
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &result); err != nil {
		return Result{}, fmt.Errorf("解析模型输出失败: %w", err)
	}
	return result, nil
}

// truncate 截断超长文本，控制送入模型的上下文长度。
func truncate(text string) string {
	runes := []rune(text)
	if len(runes) <= maxDocumentChars {
		return text
	}
	return string(runes[:maxDocumentChars])
}

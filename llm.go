package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type GeminiAnalyzer struct {
	model  string
	apiKey string
	client *http.Client
}

type GroqAnalyzer struct {
	model   string
	baseURL string
	apiKey  string
	client  *http.Client
}

type MockAnalyzer struct{}

func NewAnalyzer(cfg Config) Analyzer {
	switch cfg.LLMProvider {
	case "gemini":
		if cfg.GeminiAPIKey != "" {
			return NewGeminiAnalyzer(cfg.GeminiAPIKey, cfg.GeminiModel)
		}
	case "groq":
		if cfg.GroqAPIKey != "" {
			return NewGroqAnalyzer(cfg.GroqAPIKey, cfg.GroqModel, cfg.GroqBaseURL)
		}
	}

	return &MockAnalyzer{}
}

func NewGeminiAnalyzer(apiKey, model string) *GeminiAnalyzer {
	return &GeminiAnalyzer{
		model:  defaultString(model, "gemini-1.5-flash"),
		apiKey: apiKey,
		client: &http.Client{Timeout: 45 * time.Second},
	}
}

func NewGroqAnalyzer(apiKey, model, baseURL string) *GroqAnalyzer {
	return &GroqAnalyzer{
		model:   defaultString(model, "llama-3.1-8b-instant"),
		baseURL: defaultString(baseURL, "https://api.groq.com/openai/v1/chat/completions"),
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 45 * time.Second},
	}
}

func (g *GeminiAnalyzer) Name() string {
	return "gemini:" + g.model
}

func (g *GroqAnalyzer) Name() string {
	return "groq:" + g.model
}

func (m *MockAnalyzer) Name() string {
	return "mock:heuristic"
}

func (g *GeminiAnalyzer) AnalyzeText(ctx context.Context, post ScrapedPost) (*Insight, error) {
	if strings.TrimSpace(g.apiKey) == "" {
		return nil, fmt.Errorf("missing GEMINI_API_KEY")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)

	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": buildAnalysisPrompt(post)},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"responseMimeType": "application/json",
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from gemini")
	}

	return parseInsightPayload(geminiResp.Candidates[0].Content.Parts[0].Text)
}

func (g *GroqAnalyzer) AnalyzeText(ctx context.Context, post ScrapedPost) (*Insight, error) {
	if strings.TrimSpace(g.apiKey) == "" {
		return nil, fmt.Errorf("missing GROQ_API_KEY")
	}

	requestBody := map[string]interface{}{
		"model": g.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": `你是一個資深 SaaS 產品經理，回覆必須是 JSON，不能有任何額外文字。`,
			},
			{
				"role":    "user",
				"content": buildAnalysisPrompt(post),
			},
		},
		"temperature": 0.2,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var groqResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return nil, fmt.Errorf("failed to decode groq response: %w", err)
	}
	if len(groqResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from groq")
	}

	return parseInsightPayload(groqResp.Choices[0].Message.Content)
}

func (m *MockAnalyzer) AnalyzeText(_ context.Context, post ScrapedPost) (*Insight, error) {
	text := strings.ToLower(strings.Join([]string{post.Title, post.Content}, " "))
	score := 5
	workaround := "Users are piecing together manual steps, generic tools, or ad-hoc habits."
	feasibility := "High: likely solvable with software automation and workflow design."

	if strings.Contains(text, "manual") || strings.Contains(text, "手動") || strings.Contains(text, "copy paste") {
		score += 2
		workaround = "Users are handling the task manually, often with copy-paste, spreadsheets, or repeated clicks."
	}
	if strings.Contains(text, "spreadsheet") || strings.Contains(text, "excel") {
		score++
	}
	if strings.Contains(text, "slow") || strings.Contains(text, "好累") || strings.Contains(text, "too many features") {
		score++
	}
	if strings.Contains(text, "hardware") || strings.Contains(text, "door") || strings.Contains(text, "配送") {
		feasibility = "Medium: software can help, but operations or hardware may still be involved."
	}
	if score > 10 {
		score = 10
	}

	painPoint := normalizeText(post.Title)
	if painPoint == "" {
		painPoint = firstSentence(post.Content)
	}
	if painPoint == "" {
		painPoint = "Users are struggling with a repetitive workflow that current tools do not handle well."
	}

	explicit := containsExplicitSignals(text)

	return &Insight{
		CorePainPoint:       painPoint,
		CurrentWorkaround:   workaround,
		CommercialPotential: score,
		SaaSFeasibility:     feasibility,
		IsExplicitContent:   explicit,
	}, nil
}

func buildAnalysisPrompt(post ScrapedPost) string {
	return fmt.Sprintf(`你現在是一個資深 SaaS 產品經理。請分析以下文本，並嚴格輸出 JSON。

你要輸出的 JSON keys 只能有：
- core_pain_point
- current_workaround
- commercial_potential
- saas_feasibility
- is_explicit_content

分析標準：
1. 識別核心痛點：用戶實際在抱怨什麼？
2. 判斷目前 workaround：使用者現在如何笨拙地撐過去？
3. 商業化潛力 1-10：依市場規模、痛感強度、解法清晰度評分。
4. SaaS 實作可行性：請用一句話說明是否適合軟體化，若需要硬體或實體服務也要直接說。
5. 合規檢查：若涉及色情、未成年、暴力、毒品、詐騙、非法交易等內容，is_explicit_content 設為 true。

來源資訊：
platform: %s
channel: %s
content_kind: %s
title: %s
author: %s
url: %s

文本內容：
%s`, post.Platform, post.Channel, post.ContentKind, post.Title, post.Author, post.URL, post.Content)
}

func parseInsightPayload(raw string) (*Insight, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var insight Insight
	if err := json.Unmarshal([]byte(raw), &insight); err != nil {
		return nil, fmt.Errorf("failed to parse insight json: %w; raw=%s", err, raw)
	}

	if insight.CommercialPotential < 1 {
		insight.CommercialPotential = 1
	}
	if insight.CommercialPotential > 10 {
		insight.CommercialPotential = 10
	}
	if insight.CorePainPoint == "" {
		insight.CorePainPoint = "The source describes a frustrating workflow that existing tools are not solving cleanly."
	}
	if insight.CurrentWorkaround == "" {
		insight.CurrentWorkaround = "Users are relying on manual steps or generic tools as a workaround."
	}
	if insight.SaaSFeasibility == "" {
		insight.SaaSFeasibility = "Needs validation."
	}

	return &insight, nil
}

func firstSentence(text string) string {
	text = normalizeText(text)
	if text == "" {
		return ""
	}
	if idx := strings.IndexAny(text, ".!?。！？"); idx > 0 {
		return strings.TrimSpace(text[:idx+1])
	}
	if len(text) > 140 {
		return strings.TrimSpace(text[:140]) + "..."
	}
	return text
}

func containsExplicitSignals(text string) bool {
	signals := []string{"nsfw", "porn", "escort", "drug", "scam", "illegal", "裸體", "毒品", "詐騙", "成人"}
	for _, signal := range signals {
		if strings.Contains(text, signal) {
			return true
		}
	}
	return false
}

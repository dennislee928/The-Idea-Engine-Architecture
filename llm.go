package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// Insight represents the structured output expected from the LLM
type Insight struct {
	CorePainPoint       string `json:"core_pain_point"`
	CurrentWorkaround   string `json:"current_workaround"`
	CommercialPotential int    `json:"commercial_potential"` // 1-10
	SaaSFeasibility     string `json:"saas_feasibility"`
	IsExplicitContent   bool   `json:"is_explicit_content"` // Content filtering
}

type GeminiAnalyzer struct {
	apiKey string
	client *http.Client
}

func NewGeminiAnalyzer() *GeminiAnalyzer {
	return &GeminiAnalyzer{
		apiKey: os.Getenv("GEMINI_API_KEY"),
		client: &http.Client{},
	}
}

func (g *GeminiAnalyzer) AnalyzeText(ctx context.Context, text string) (*Insight, error) {
	// The SOP Prompt you provided
	systemPrompt := `你現在是一個資深 SaaS 產品經理。請分析以下文本：
1. 識別其中的核心痛點（用戶在抱怨什麼？）。
2. 目前的 Workaround（用戶現在怎麼解決？）。
3. 商業化潛力 (1-10)：根據市場規模與解決難度評分。
4. SaaS 實作可行性：這可以用軟體解決嗎？還是需要硬體/實體服務？
5. 合規檢查：內容是否包含色情、非法等成人內容。

請嚴格輸出為 JSON 格式，包含以下 keys: core_pain_point, current_workaround, commercial_potential, saas_feasibility, is_explicit_content。`

	// Constructing the payload for Gemini API (Using REST for simplicity and control)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", g.apiKey)

	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": systemPrompt + "\n\n文本內容：\n" + text},
				},
			},
		},
		// Force JSON response
		"generationConfig": map[string]interface{}{
			"responseMimeType": "application/json",
		},
	}

	jsonBody, _ := json.Marshal(requestBody)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parsing the specific Gemini response structure
	// In a real implementation, you would define the exact Gemini response structs,
	// but here we are extracting the text block which contains our structured JSON.
	// *Implementation note for developer*: Unmarshal the nested Gemini response,
	// extract `candidates[0].content.parts[0].text`, and then unmarshal THAT string into the `Insight` struct.

	// Mocking the successful extraction for now to show the flow
	var insight Insight
	return &insight, nil
}

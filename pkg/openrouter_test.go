package pkg

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"
)

func GetOpenRouterClient() *OpenRouter {
	conf := LoadOpenRouterConfFromEnv()

	conf.Debug = false

	return NewOpenRouter(conf)
}

func TestOpenRouter_HandleProxyJSON(t *testing.T) {
	openrouter := GetOpenRouterClient()


	// 創建一個測試請求
	req := httptest.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewBufferString(`{
	"model": "deepseek-reasoner",
	"stream": false,
	"messages": [{"role": "user", "content": "明天的前天，是昨天的后天吗？"}]
}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")


	// 創建一個響應記錄器
	w := httptest.NewRecorder()

	openrouter.HandleProxy(w, req)

	// 檢查響應
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	} else {
		t.Logf("status: %d", resp.StatusCode)
		t.Logf("headers: %v", resp.Header)
		respData, err := io.ReadAll(resp.Body)
		if err == nil {
			t.Logf("body: %s", string(respData))
		}
	}
}

func TestOpenRouter_HandleProxyStream(t *testing.T) {
	openrouter := GetOpenRouterClient()


	// 創建一個測試請求
	req := httptest.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewBufferString(`{
	"model": "deepseek-reasoner",
	"stream": true,
	"messages": [{"role": "user", "content": "明天的前天，是昨天的后天吗？"}]
}`))
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")


	// 創建一個響應記錄器
	w := httptest.NewRecorder()

	openrouter.HandleProxy(w, req)

	// 檢查響應
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	} else {
		t.Logf("status: %d", resp.StatusCode)
		t.Logf("headers: %v", resp.Header)
		respData, err := io.ReadAll(resp.Body)
		if err == nil {
			t.Logf("body: %s", string(respData))
		}
	}
}
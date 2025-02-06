package pkg

import (
	"bytes"
	"strings"
	"testing"
)

func GetOpenRouterClient() *OpenRouter {
	conf := LoadOpenRouterConfFromEnv()

	conf.Debug = false

	return NewOpenRouter(conf)
}

func TestOpenRouter_MessageCompletionJSON(t *testing.T) {
	openrouter := GetOpenRouterClient()

	json := strings.NewReader(`{
	"model": "deepseek-reasoner",
	"stream": true,
	"messages": [{"role": "user", "content": "明天的前天，是昨天的后天吗？"}]
}`)

	params, options, err := openrouter.GetParamsFromRequestBody(json)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	resp, err := openrouter.MessageCompletionRaw(params, options)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	t.Logf("is stream: %v", resp.IsStream())

	if !resp.IsStream() {
		var b bytes.Buffer
		_, err = b.ReadFrom(resp.GetResponse())
		if err != nil {
			t.Error(err)
			t.Fail()
			return
		}
		t.Log(b.String())
	} else {
		for event := range resp.GetEvents() {
			t.Logf("%+v", event)
		}
	}

}
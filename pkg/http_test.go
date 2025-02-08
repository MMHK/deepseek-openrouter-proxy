package pkg

import (
	"deepseek-openrouter-proxy/tests"
	"testing"
)

func TestHTTPService_Start(t *testing.T) {
	conf := &Config{}
	conf.MarginWithENV()

	t.Log(tests.ToJSON(conf))

	http := NewHttpService(conf)
	http.Start()
}

package pkg

import (
	"deepseek-openrouter-proxy/tests"
	"testing"
)

func TestConfig_MarginWithENV(t *testing.T) {
	conf := &Config{}
	conf.MarginWithENV()

	t.Logf("%+v", conf)
	t.Log(tests.ToJSON(conf))
	t.Log("PASS")
}

func Test_SaveConfig(t *testing.T) {

	conf := &Config{}
	conf.MarginWithENV()

	configPath := tests.GetLocalPath("../config.json")
	err := conf.Save(configPath)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	t.Log("PASS")
}

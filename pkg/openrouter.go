package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

type ISSEDecoder interface {
	GetBytes() []byte
}

type StreamChunk struct {
	Chunk  string
}

func (this *StreamChunk) GetBytes() []byte {
	return []byte(this.Chunk)
}

type IStreamableResponse interface {
	IsStream() bool
	GetResponse() io.Reader
	GetEvents() <-chan ISSEDecoder
}

type StdRequestParams struct {
	Stream bool   `json:"stream,omitempty"`
	Model  string `json:"model,required"`
}

type MessageCompleteResponse struct {
	stream   bool
	Response io.Reader
	Events   <-chan ISSEDecoder
}

func (this *MessageCompleteResponse) IsStream() bool {
	return this.stream
}

func (this *MessageCompleteResponse) GetResponse() io.Reader {
	return this.Response
}

func (this *MessageCompleteResponse) GetEvents() <-chan ISSEDecoder {
	return this.Events
}

type OpenRouterConf struct {
	BaseURL            string            `json:"base_url,omitempty"`
	APIKey             string            `json:"api_key,omitempty"`
	ModelMappings      map[string]string `json:"model_mappings"`
	EnableOutputReason bool              `json:"enable_reason,omitempty"`
	RankingsTitle      string            `json:"rankings_title,omitempty"`
	RankingsURL        string            `json:"rankings_url,omitempty"`
	Debug              bool              `json:"-"`
}

func LoadOpenRouterConfFromEnv() (*OpenRouterConf) {
	conf := &OpenRouterConf{}
	conf.BaseURL = os.Getenv("OPENROUTER_BASE_URL")
	conf.APIKey = os.Getenv("OPENROUTER_API_KEY")
	conf.EnableOutputReason = os.Getenv("OPENROUTER_ENABLE_OUTPUT_REASON") == "true"

	conf.RankingsTitle = os.Getenv("OPENROUTER_RANKINGS_TITLE")
	conf.RankingsURL = os.Getenv("OPENROUTER_RANKINGS_URL")

	modelMappingsJSON := os.Getenv("OPENROUTER_MODEL_MAPPINGS")
	if len(modelMappingsJSON) > 0 {
		err := json.Unmarshal([]byte(modelMappingsJSON), &conf.ModelMappings)
		if err != nil {
			Log.Error(err)
		}
	}

	return conf
}

type OpenRouter struct {
	*openai.Client
	*OpenRouterConf
}

func NewOpenRouter(conf *OpenRouterConf) (*OpenRouter) {
	client := openai.NewClient(option.WithAPIKey(conf.APIKey),
	option.WithBaseURL(conf.BaseURL))

	return &OpenRouter{
		Client: client,
		OpenRouterConf: conf,
	}
}

func (this *OpenRouter) GetModelMappings(source string) (string, error) {
	if len(this.ModelMappings) > 0 {
		if target, ok := this.ModelMappings[source]; ok {
			return target, nil
		}
	}

	return source, errors.New(fmt.Sprintf("model %s not found in model mappings", source))
}

func (this *OpenRouter) GetParamsFromRequestBody(reader io.Reader) (params *StdRequestParams, RequestOptions []option.RequestOption, err error) {
	params = &StdRequestParams{}
	RequestOptions = []option.RequestOption{}

	var buf bytes.Buffer
	teeReader := io.TeeReader(reader, &buf)

	decoder := json.NewDecoder(teeReader)

	err = decoder.Decode(params)
	if err != nil {
		Log.Error(err)
		return nil, RequestOptions, err
	}

	RequestOptions = append(RequestOptions, option.WithRequestBody("application/json", &buf))

	return params, RequestOptions, nil
}

// 自定义的 RoundTripper 用于记录请求和响应
type loggingRoundTripper struct {
	wrapped http.RoundTripper
}

func (l loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// 记录请求
	reqDump, _ := httputil.DumpRequestOut(req, true)
	Log.Infof("Request:\n%s", string(reqDump))

	// 发送请求
	resp, err := l.wrapped.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// 记录响应
	respDump, _ := httputil.DumpResponse(resp, true)
	Log.Infof("Response:\n%s", string(respDump))

	// 重要：我们需要重新创建响应体，因为 DumpResponse 会消耗它
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	return resp, nil
}

func (this *OpenRouter) MessageCompletionRaw(params *StdRequestParams, RequestOptions []option.RequestOption) (IStreamableResponse, error) {
	ctx := context.Background()

	param := openai.ChatCompletionNewParams{
		Model: openai.F(params.Model),
	}

	model, err := this.GetModelMappings(param.Model.String())
	if err != nil {
		Log.Error(err)
		return nil, err
	}

	param.Model = openai.F(model)



	RequestOptions = append(RequestOptions,
		option.WithHeader("X-Title", this.RankingsTitle),
		option.WithHeader("HTTP-Referer", this.RankingsURL),
		option.WithJSONSet("model", model))

	if this.OpenRouterConf.Debug {
		// 创建自定义的 http.Client
		customClient := &http.Client{
			Transport: &loggingRoundTripper{
				wrapped: http.DefaultTransport,
			},
		}

		RequestOptions = append(RequestOptions, option.WithHTTPClient(customClient))
	}

	if this.OpenRouterConf.EnableOutputReason {
		RequestOptions = append(RequestOptions, option.WithJSONSet("include_reasoning", true))
	}

	if params.Stream {
		streamResp := this.Client.Chat.Completions.NewStreaming(ctx, param, RequestOptions...)

		eventQueue := make(chan ISSEDecoder, 10)

		go func() {
			defer close(eventQueue)

			for streamResp.Next() {
				chunk := streamResp.Current()
				eventQueue <- &StreamChunk{
					Chunk: strings.ReplaceAll(chunk.JSON.RawJSON(), `"reasoning"`, `"reasoning_content"`),
				}
			}
		}()

		return &MessageCompleteResponse{
			stream:   true,
			Events:   eventQueue,
		}, nil
	}

	completion, err := this.Client.Chat.Completions.New(ctx, param, RequestOptions...)
	if err != nil {
		Log.Error(err)
		return nil, err
	}

	if this.OpenRouterConf.EnableOutputReason {
		output := completion.JSON.RawJSON();
		output = strings.ReplaceAll(output, `"reasoning"`, `"reasoning_content"`)

		return &MessageCompleteResponse{
			stream:   false,
			Response: strings.NewReader(output),
		}, nil
	}

	return &MessageCompleteResponse{
		stream:   false,
		Response: strings.NewReader(completion.JSON.RawJSON()),
	}, nil
}




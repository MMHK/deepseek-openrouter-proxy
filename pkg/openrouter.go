package pkg

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)


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
	*OpenRouterConf
	proxy  *httputil.ReverseProxy
}

func NewOpenRouter(conf *OpenRouterConf) (*OpenRouter) {
	targetURl, err := url.Parse(conf.BaseURL)
	if err != nil {
		Log.Error(err)
	}

	return &OpenRouter{
		OpenRouterConf: conf,
		proxy:  httputil.NewSingleHostReverseProxy(targetURl),
	}
}

func (this *OpenRouter) GetModelMappings(source string) (string, error) {
	if len(this.ModelMappings) > 0 {
		if target, ok := this.OpenRouterConf.ModelMappings[source]; ok {
			return target, nil
		}
	}

	return source, errors.New(fmt.Sprintf("model %s not found in model mappings", source))
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
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return resp, nil
}

type responseWriter struct {
	http.ResponseWriter
	buf *bytes.Buffer
	status int
	header http.Header
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.buf.Write(p)
}

func (rw *responseWriter) Header() http.Header {
	return rw.header
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.status = statusCode
}

func (rw *responseWriter) Flush() {
	rw.ResponseWriter.(http.Flusher).Flush()
}

func (this *OpenRouter) IsAcceptStream(req *http.Request) (*http.Request, bool) {
	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "json") {
		var buf bytes.Buffer
		reader := io.TeeReader(req.Body, &buf)
		type streamWrapper struct {
			Stream bool `json:"stream"`
		}
		decoder := json.NewDecoder(reader)
		isStream := new(streamWrapper)
		err := decoder.Decode(isStream)
		cloneReq := &http.Request{
			Method: req.Method,
			URL:    req.URL,
			Proto:  req.Proto,
			Header: req.Header.Clone(),
			Body:   io.NopCloser(bytes.NewBuffer(buf.Bytes())),
		}
		if err != nil {
			Log.Error(err)
			return cloneReq, false
		}

		return cloneReq, isStream.Stream
	}

	return req, false
}

func (this *OpenRouter) HandleProxy(w http.ResponseWriter, r *http.Request) {
	cloneReq, isStream := this.IsAcceptStream(r)

	if isStream {
		// 複製原始請求的 headers
		this.modifyRequest(cloneReq)

		var (
			resp *http.Response
			err  error
		)
		// 發送請求到目標服務器
		if this.Debug {
			reqDump, _ := httputil.DumpRequestOut(cloneReq, true)
			Log.Infof("Request:\n%s", string(reqDump))
		}

		resp, err = http.DefaultClient.Do(cloneReq)
		if err != nil {
			Log.Error(err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if err := this.handleSSEStream(w, resp); err != nil {
			Log.Error(err)
		}
		return
	}

	this.proxy.ModifyResponse = this.modifyResponse
	this.proxy.Director = this.modifyRequest

	customWriter := &responseWriter{
		ResponseWriter: w,
		buf:            new(bytes.Buffer),
		header:         make(http.Header),
	}

	if this.Debug {
		this.proxy.Transport = &loggingRoundTripper{
			wrapped: http.DefaultTransport,
		}
	}

	this.proxy.ServeHTTP(customWriter, cloneReq)

	// 寫入修改後的響應
	w.WriteHeader(customWriter.status)
	headers := customWriter.Header()
	for k, v := range headers {
		w.Header()[k] = v
	}
	_, err := w.Write(customWriter.buf.Bytes())
	if err != nil {
		Log.Error(err)
	}
}

func (this *OpenRouter) modifyRequest(req *http.Request) {
	Log.Infof("modifyRequest: %s %s", req.Method, req.URL.String())

	targetURL, err := url.Parse(this.BaseURL)
	if err != nil {
		Log.Error(err)
		return
	}
	req.URL.Host = targetURL.Host
	req.URL.Scheme = targetURL.Scheme
	req.URL.Path = filepath.ToSlash(filepath.Join(targetURL.Path, req.URL.Path))
	req.Host = targetURL.Host

	if this.OpenRouterConf.RankingsTitle != "" {
		req.Header.Set("X-Title", this.RankingsTitle)
	}

	if this.OpenRouterConf.RankingsURL != "" {
		req.Header.Set("HTTP-Referer", this.RankingsURL)
	}

	if this.OpenRouterConf.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", this.APIKey))
	}

	contentType := req.Header.Get("Content-Type")
	//Log.Infof("Content-Type: %s", contentType)
	if strings.Contains(contentType, "json") {
		var body map[string]interface{}
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&body)
		if err != nil {
			Log.Error(err)
			return
		}
		//Log.Infof("req body: %+v", body)
		targetModel := ""
		if srcModel, ok := body["model"]; ok {
			//Log.Infof("src model: %s", srcModel)
			if model, ok := srcModel.(string); ok {
				targetModel, err = this.GetModelMappings(model)
				if err != nil {
					Log.Error(err)
					return
				}
				//Log.Infof("mapped model: %s", targetModel)
			}
		}

		if this.EnableOutputReason {
			body["include_reasoning"] = true
		}

		if targetModel != "" {
			body["model"] = targetModel
		}

		newBody, err := json.Marshal(body)
		if err != nil {
			Log.Error(err)
			return
		}

		req.Body = io.NopCloser(bytes.NewBuffer(newBody))
		req.ContentLength = int64(len(newBody))
		req.Header.Set("Content-Length", strconv.Itoa(len(newBody)))
	}
}

func (this *OpenRouter) modifyResponse(res *http.Response) error {
	// 檢查內容類型
	contentType := res.Header.Get("Content-Type")

	if strings.Contains(contentType, "json") {
		// 處理 JSON
		return this.handleJSON(res)
	}

	// 其他類型不做修改
	return nil
}

func (this *OpenRouter) handleSSEStream(w http.ResponseWriter, res *http.Response) error {
	// 設置 SSE 相關的 headers
	for k, v := range res.Header {
		w.Header()[k] = v
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	if this.Debug {
		Log.Infof("handleSSEStream: %s", res.Header.Get("Content-Type"))
	}


	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// 這裡可以修改 SSE 數據
		if this.EnableOutputReason && strings.HasPrefix(line, "data:") {
			line = strings.ReplaceAll(line, `"reasoning"`, `"reasoning_content"`)
		}

		// 寫入修改後的行並立即刷新
		fmt.Fprintf(w, "%s\n", line)
		if this.Debug {
			Log.Infof("SSE: %s\n", line)
		}
		flusher.Flush()
	}

	if err := scanner.Err(); err != nil {
		Log.Error(err)
		return err
	}

	return nil
}

func (this *OpenRouter) handleJSON(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		Log.Error(err)
		return err
	}

	newBody := bytes.ReplaceAll(body, []byte(`"reasoning"`), []byte(`"reasoning_content"`))

	// 替換原始 body
	resp.Body = io.NopCloser(bytes.NewBuffer(newBody))
	resp.ContentLength = int64(len(newBody))
	resp.Header.Set("Content-Length", strconv.Itoa(len(newBody)))

	return nil
}




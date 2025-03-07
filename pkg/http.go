package pkg

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

type HttpConfig struct {
	Listen  string `json:"listen,omitempty"`
	WebRoot string `json:"web_root,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
}

type HTTPService struct {
	conf *Config
}

type APIError struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
}

type APIStandardError struct {
	Type  string    `json:"type,omitempty"`
	Error *APIError `json:"error,omitempty"`
}

func NewHttpService(conf *Config) *HTTPService {
	return &HTTPService{
		conf: conf,
	}
}

func (this *HTTPService) RedirectSwagger(writer http.ResponseWriter, request *http.Request) {
	http.Redirect(writer, request, "/swagger/", 301)
}

func (this *HTTPService) NotFoundHandle(writer http.ResponseWriter, request *http.Request) {
	server_error := &APIStandardError{Type: "error", Error: &APIError{
		Type:    "error",
		Message: "not found",
	}}
	json_str, _ := json.Marshal(server_error)
	http.Error(writer, string(json_str), 404)
}

func (this *HTTPService) ResponseError(err error, writer http.ResponseWriter) {
	server_error := &APIStandardError{Type: "error", Error: &APIError{
		Type:    "invalid_request_error",
		Message: err.Error(),
	}}
	json_str, _ := json.Marshal(server_error)
	http.Error(writer, string(json_str), 200)
}

func (this *HTTPService) ResponseJSON(source interface{}, writer http.ResponseWriter) {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)

	writer.Header().Set("Content-Type", "application/json")
	err := encoder.Encode(source)
	if err != nil {
		this.ResponseError(err, writer)
	}
}

// APIKeyMiddleware 验证 API Key 的中间件
func (this *HTTPService) APIKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		Log.Debug("APIKeyMiddleware")
		APIKey := this.conf.APIKey
		Log.Debugf("APIKeyMiddleware: %s", APIKey)
		if APIKey == "" {
			next.ServeHTTP(writer, request)
			return
		}
		apiKey := request.Header.Get("Authorization")
		apiKey = strings.Replace(apiKey, "Bearer ", "", -1)
		Log.Debugf("API key in header: %s", apiKey)
		if apiKey == "" {
			this.ResponseError(fmt.Errorf("invalid api key"), writer)
			return
		}

		// 这里可以添加更多的 API Key 验证逻辑
		if apiKey != APIKey {
			this.ResponseError(fmt.Errorf("Invalid API key"), writer)
			return
		}

		next.ServeHTTP(writer, request)
	})
}


func (this *HTTPService) Start() {
	rHandler := mux.NewRouter()

	// 需要 API Key 的路由
	apiRouter := rHandler.PathPrefix("/api").Subrouter()
	apiRouter.Use(this.APIKeyMiddleware)

	conf := LoadOpenRouterConfFromEnv()
	openRouter := NewOpenRouter(conf)

	deepseekRouter := apiRouter.PathPrefix("/deepseek").Subrouter()
	deepseekRouter.PathPrefix("/").
		Handler(http.StripPrefix("/api/deepseek", http.HandlerFunc(openRouter.HandleProxy)))

	rHandler.HandleFunc("/", this.RedirectSwagger)
	rHandler.PathPrefix("/").Handler(http.StripPrefix("/",
		http.FileServer(http.Dir(fmt.Sprintf("%s", this.conf.WebRoot)))))
	rHandler.NotFoundHandler = http.HandlerFunc(this.NotFoundHandle)

	Log.Info("http service starting")
	Log.Infof("Please open http://%s\n", this.conf.Listen)
	err := http.ListenAndServe(this.conf.Listen, rHandler)
	if err != nil {
		Log.Error(err)
	}
}

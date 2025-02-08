# deepseek-openrouter-proxy

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/mmhk/deepseek-openrouter-proxy)](https://goreportcard.com/report/github.com/mmhk/deepseek-openrouter-proxy)
[![Docker Image](https://img.shields.io/docker/pulls/mmhk/deepseek-openrouter-proxy.svg)](https://hub.docker.com/r/mmhk/deepseek-openrouter-proxy)


deepseek-openrouter-proxy is a Golang-based API converter that transforms OpenRouter API (similar to OpenAI) into DeepSeek's official API style. This project aims to provide developers with a seamless interface for converting between OpenRouter and DeepSeek APIs, enhancing the interoperability and flexibility of AI model interactions.

## 配置

项目使用环境变量进行配置。请参考以下示例创建 `.env` 文件，并根据需要修改配置项：

```dotenv
WEB_ROOT=/api 
HTTP_LISTEN=:8080 
API_KEY=your_api_key_here 
OPENROUTER_BASE_URL=https://openrouter.ai/api/v1
OPENROUTER_API_KEY=your_openrouter_api_key_here 
OPENROUTER_ENABLE_OUTPUT_REASON=true 
OPENROUTER_MODEL_MAPPINGS={"deepseek-reasoner":"deepseek/deepseek-r1","deepseek-chat":"deepseek/deepseek-chat"}
OPENROUTER_RANKINGS_TITLE=deepseek-openrouter-proxy
OPENROUTER_RANKINGS_URL=https://github.com/mmhk/deepseek-openrouter-proxy
LOG_LEVEL=INFO
```

### 配置项说明

- **WEB_ROOT**: API 的根路径，默认为 `/api`。
- **HTTP_LISTEN**: HTTP 服务器监听地址和端口，默认为 `:8080`。
- **API_KEY**: 项目的 API 密钥，请替换为实际的密钥。
- **OPENROUTER_BASE_URL**: OpenRouter API 的基础 URL，请替换为实际的 URL。
- **OPENROUTER_API_KEY**: OpenRouter API 的密钥，请替换为实际的密钥。
- **OPENROUTER_ENABLE_OUTPUT_REASON**: 是否启用输出推理过程，默认为 `true`。
- **OPENROUTER_MODEL_MAPPINGS**: OpenRouter 模型到 DeepSeek 模型的映射，格式为 JSON
- **OPENROUTER_RANKINGS_TITLE**: OpenRouter 用于跟踪调用链，记录API 求情的标题，默认为 `deepseek-openrouter-proxy`。
- **OPENROUTER_RANKINGS_URL**: OpenRouter 用于跟踪调用链，记录API 求情的 URL，默认为 `https://github.com/mmhk/deepseek-openrouter-proxy`。
- **LOG_LEVEL**: 日志级别，默认为 `INFO`，可选值包括 `DEBUG`, `INFO`, `WARN`, `ERROR`。

## 使用方法

根据 `docker-compose.yml` 文件中的配置，以下是完善后的使用方法：

### 使用方法

1. **安装 Docker 和 Docker Compose**
    - 确保你的系统上已经安装了 Docker 和 Docker Compose。如果没有安装，可以从 [Docker 官方网站](https://www.docker.com/products/docker-desktop) 下载并安装。

2. **克隆项目仓库**
    - 使用 Git 克隆项目仓库到本地：
      ```bash
      git clone https://github.com/mmhk/deepseek-openrouter-proxy.git
      cd deepseek-openrouter-proxy
      ```


3. **创建 `.env` 文件**
    - 根据 `README.md` 中的配置项说明，创建一个 `.env` 文件，并根据实际情况填写必要的环境变量：
      ```dotenv
      API_KEY=your_api_key_here
      OPENROUTER_API_KEY=your_openrouter_api_key_here
      AZURE_API_KEY=your_azure_api_key_here
      ```


4. **启动服务**
    - 使用 Docker Compose 启动服务：
      ```bash
      docker-compose up -d
      ```

    - 这将以后台模式启动 `deepseek-openrouter-proxy` 和 `chatgpt-next` 两个服务。

5. **验证服务是否正常运行**
    - 检查服务是否正常运行：
      ```bash
      docker-compose ps
      ```

    - 你应该能看到两个服务的状态为 `Up`。

6. **访问服务**
    - 打开浏览器，访问 `http://localhost:8809` 可以查看 `deepseek-openrouter-proxy` 的状态。
    - 访问 `http://localhost:4001` 可以使用 `chatgpt-next` 服务。

7. **停止服务**
    - 如果需要停止服务，可以使用以下命令：
      ```bash
      docker-compose down
      ```


通过以上步骤，你可以成功部署并运行 `deepseek-openrouter-proxy` 和 `chatgpt-next` 服务。

## 许可证

本项目遵循 [Apache License 2.0](https://opensource.org/licenses/Apache-2.0) 许可证。
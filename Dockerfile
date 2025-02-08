FROM golang:1.21-alpine as builder

# Add Maintainer Info
LABEL maintainer="Sam Zhou <sam@mixmedia.com>"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go version \
 && export GO111MODULE=on \
 && export GOPROXY=https://goproxy.io,direct \
 && go mod vendor \
 && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o deepseek-openrouter-proxy


######## Start a new stage from scratch #######
FROM alpine:latest

RUN apk add --update libintl \
    && apk add --no-cache ca-certificates tzdata dumb-init \
    && apk add --virtual build_deps gettext  \
    && cp /usr/bin/envsubst /usr/local/bin/envsubst \
    && apk del build_deps

WORKDIR /app


# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/deepseek-openrouter-proxy .
COPY --from=builder /app/webroot .

ENV HTTP_LISTEN=0.0.0.0:8809 \
 WEB_ROOT=/app/webroot \
 API_KEY= \
 OPENROUTER_BASE_URL=https://openrouter.ai/api/v1 \
 OPENROUTER_API_KEY= \
 OPENROUTER_ENABLE_OUTPUT_REASON=true \
 OPENROUTER_MODEL_MAPPINGS={"deepseek-reasoner":"deepseek/deepseek-r1","deepseek-chat":"deepseek/deepseek-chat"} \
 OPENROUTER_RANKINGS_TITLE=deepseek-openrouter-proxy \
 OPENROUTER_RANKINGS_URL=https://github.com/mmhk/deepseek-openrouter-proxy \
 LOG_LEVEL=INFO

EXPOSE 8809

ENTRYPOINT ["dumb-init", "--"]

CMD /app/deepseek-openrouter-proxy


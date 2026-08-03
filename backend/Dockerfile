# --- 构建 stage ---
FROM golang:1.24.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/oh-my-stock .

# --- 运行 stage ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app
ENV TZ=Asia/Shanghai
USER app
WORKDIR /app
COPY --from=build /out/oh-my-stock /app/oh-my-stock
COPY config.json /app/config.json
COPY docs /app/docs
EXPOSE 3003
ENTRYPOINT ["/app/oh-my-stock"]

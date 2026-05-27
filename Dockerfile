FROM oven/bun:1 AS frontend

WORKDIR /web

COPY web/package.json web/bun.lock* ./

RUN bun install --frozen-lockfile

COPY web/ .

RUN bun run build

FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

COPY --from=frontend /web/dist/index.html internal/managementasset/static/management.html

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN mkdir -p internal/managementasset/static && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.BuildDate=${BUILD_DATE}'" -o ./LLMHub ./cmd/server/

FROM alpine:3.23

RUN apk add --no-cache tzdata

RUN mkdir /LLMHub

COPY --from=builder ./app/LLMHub /LLMHub/LLMHub

COPY config.example.yaml /LLMHub/config.example.yaml

WORKDIR /LLMHub

EXPOSE 8317

ENV TZ=Asia/Shanghai

RUN cp /usr/share/zoneinfo/${TZ} /etc/localtime && echo "${TZ}" > /etc/timezone

CMD ["./LLMHub"]

FROM golang:1.22-alpine3.19 AS builder

ENV GOPROXY=https://goproxy.cn

WORKDIR /app

COPY go.mod ./go.mod

COPY go.sum ./go.sum

RUN go mod download

COPY . /app

RUN go build -ldflags "-w -s -buildid=" -trimpath -o server

FROM zenika/alpine-chrome:with-node

USER root

RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories

RUN apk add --no-cache \
    git

RUN npm config set registry https://registry.npmmirror.com

RUN npm install -g single-file-cli

WORKDIR /app

COPY --from=builder /app/server /app/server

RUN chmod +x /app/server

ENTRYPOINT [ "/app/server" ]
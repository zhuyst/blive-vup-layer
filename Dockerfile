FROM node:20.12.2-buster AS BUILDER1

WORKDIR /usr/src/app

COPY ./frontend/ .

RUN npm install && npm run build

FROM golang:1.20.14-bullseye AS BUILDER2

WORKDIR /usr/src/app

ENV GOPROXY http://goproxy.cn

COPY . .

RUN CGO_ENABLED=0 go build -o blive-vup-layer

FROM debian:11.4-slim

WORKDIR /usr/src/app

ENV DEBIAN_FRONTEND noninteractive

RUN sed -i s@/[a-z]*.debian.org/@/mirrors.tuna.tsinghua.edu.cn/@g /etc/apt/sources.list && \
    apt-get update && \
    apt-get install -y ca-certificates tzdata && \
    ln -fs /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    dpkg-reconfigure -f noninteractive tzdata && \
    rm -rf /var/lib/apt/lists/*

COPY --from=BUILDER1 /usr/src/app/dist /usr/src/app/frontend/dist
COPY --from=BUILDER2 /usr/src/app/blive-vup-layer /usr/src/app/blive-vup-layer

RUN chmod +x /usr/src/app/blive-vup-layer

ENTRYPOINT ["/usr/src/app/blive-vup-layer"]
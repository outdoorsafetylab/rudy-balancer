FROM golang:1.24-alpine AS go-builder

RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
      git \
    && rm -rf /var/cache/apk/* /tmp/*

RUN mkdir -p /src/
COPY . /src/
WORKDIR /src/

ARG GIT_HASH
ARG GIT_TAG
RUN go mod tidy && go build -ldflags="-X service/version.GitHash=${GIT_HASH} -X service/version.GitTag=${GIT_TAG}" -o rudy-balancer .

FROM alpine:3.12

RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
    && rm -rf /var/cache/apk/* /tmp/*

COPY --from=go-builder /src/rudy-balancer /usr/sbin/
RUN mkdir -p /etc/rudy-balancer/
COPY config/docker.yaml /
COPY config/mirrors.yaml /
COPY webroot /webroot

EXPOSE 8080

ENTRYPOINT [ "/usr/sbin/rudy-balancer", "serve", "-c", "docker" ]

FROM golang:1.15-alpine as go-builder

RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
      curl ca-certificates protobuf \
      git make \
    && rm -rf /var/cache/apk/* /tmp/*

RUN mkdir -p /src/
COPY . /src/
WORKDIR /src/

ARG GIT_HASH
ARG GIT_TAG
RUN go build -ldflags="-X service/version.GitHash=${GIT_HASH} -X service/version.GitTag=${GIT_TAG}" -o serviced .

FROM alpine:3.12

RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
    && rm -rf /var/cache/apk/* /tmp/*

COPY --from=go-builder /src/serviced /usr/sbin/
RUN mkdir -p /etc/serviced/
COPY config/docker.yaml /
COPY config/mirrors.yaml /
COPY webroot /webroot

EXPOSE 8080

ENTRYPOINT [ "/usr/sbin/serviced", "-e", "docker", "-w", "/webroot" ]

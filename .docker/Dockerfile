FROM golang:1.15-alpine as builder

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
RUN make clean \
    && make tidy \
    && make GIT_HASH=${GIT_HASH} GIT_TAG=${GIT_TAG}

FROM alpine:3.12

RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
    && rm -rf /var/cache/apk/* /tmp/*

COPY --from=builder /src/serviced /usr/sbin/
RUN mkdir -p /etc/serviced/
COPY .docker/entrypoint.sh /
COPY config.yaml /
COPY mirrors.yaml /
COPY webroot /webroot

ENV GEOIP2_LICENSE_KEY=

EXPOSE 8080

VOLUME ["/var/lib/serviced"]

CMD ["/entrypoint.sh"]

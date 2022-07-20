FROM golang:1.18.1-alpine as builder

ARG http_proxy
ARG https_proxy
RUN apk add --no-cache make git build-base

WORKDIR /gofin-src

# copy dependency file first, avoid frequent go.mod download
COPY go.mod /gofin-src
#ENV GOPROXY https://goproxy.io,direct
RUN go mod download
RUN go install github.com/ahmetb/govvv@latest

# copy source files
COPY . /gofin-src
RUN govvv build -o jerry -tags docker -version $(go generate ./telegram/jerry) ./telegram/jerry
RUN mv ./jerry /jerry

FROM alpine:latest

# for time.LoadLocation
RUN apk add --no-cache ca-certificates tzdata libc6-compat libgcc libstdc++
# the trailing slash is a must for .json to get copied to directory /etc/jerry/
COPY --from=builder /jerry /

ENTRYPOINT ["/jerry"]

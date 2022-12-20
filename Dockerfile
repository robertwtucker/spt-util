FROM golang:1.19-bullseye AS build-env

ARG BUILD_VERSION=development
ARG BUILD_REVISION=unknown
ARG PROJECT="github.com/robertwtucker/spt-util"

WORKDIR /go/src/app
# copy module files first so that they don't need to be downloaded again if no change
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags \
  "-X ${PROJECT}/internal/config.appVersion=${BUILD_VERSION} -X ${PROJECT}/internal/config.revision=${BUILD_REVISION}" \
  -o /go/bin/app

FROM redhat/ubi8-minimal:8.7

COPY --from=build-env /go/bin/app /
COPY config/spt-util.yaml /config/spt-util.yaml
COPY tmp/deployment /deployment
CMD ["/app","--version"]

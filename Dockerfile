FROM golang:1.19-bullseye AS build-env

ARG VERSION=development
ARG REVISION=unknown
ARG PACKAGE="github.com/robertwtucker/spt-util/internal/config"

WORKDIR /go/src/app
# copy module files first so that they don't need to be downloaded again if no change
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags \
  "-X ${PACKAGE}.appVersion=${VERSION} -X ${PACKAGE}.revision=${REVISION}" \
  -o /go/bin/app

FROM redhat/ubi8-minimal:8.7

COPY --from=build-env /go/bin/app /
COPY config/spt-util.yaml /config/spt-util.yaml
COPY tmp/deployment /deployment
CMD ["/app","--version"]

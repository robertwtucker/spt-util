FROM golang:1.19 AS build-env

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

FROM gcr.io/distroless/static

COPY --from=build-env /go/bin/app /
USER nonroot:nonroot
CMD ["/app","--version"]

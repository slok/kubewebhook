FROM golang:1.10-alpine AS build-stage

ARG example

WORKDIR /go/src/github.com/slok/kubewebhook
COPY . .

RUN CGO_ENABLED=0 go build -o /bin/example --ldflags "-w -extldflags '-static'"  github.com/slok/kubewebhook/examples/${example}

# Final image.
FROM alpine:latest
RUN apk --no-cache add \
  ca-certificates
COPY --from=build-stage /bin/example /usr/local/bin/example
ENTRYPOINT ["/usr/local/bin/example"]
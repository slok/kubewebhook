FROM golang:1.10-alpine AS build-stage

WORKDIR /go/src/github.com/slok/kubewebhook
COPY . .

RUN CGO_ENABLED=0 go build -o /bin/pod-annotator --ldflags "-w -extldflags '-static'"  github.com/slok/kubewebhook/examples/pod-annotate

# Final image.
FROM alpine:latest
RUN apk --no-cache add \
  ca-certificates
COPY --from=build-stage /bin/pod-annotator /usr/local/bin/pod-annotator
ENTRYPOINT ["/usr/local/bin/pod-annotator"]
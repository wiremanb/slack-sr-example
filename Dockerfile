FROM golang:1.15-alpine AS build
RUN apk --no-cache update && \
    apk --no-cache add make ca-certificates git && \
    rm -rf /var/cache/apk/*
WORKDIR /go/src/github.com/wiremanb/slack-sr-example
COPY go.mod /go/src/github.com/wiremanb/slack-sr-example
COPY go.sum /go/src/github.com/wiremanb/slack-sr-example
RUN go mod download
COPY . ./
RUN	CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -ldflags \
    "-X github.com/wiremanb/slack-sr-example/version.Version=`git describe --tags` -X github.com/wiremanb/slack-sr-example/version.Commit=`git log -n 1 --pretty=format:"%h"`" \
    -o bin/slack-sr-example

FROM alpine:latest AS slack-sr-example
LABEL mainainer="Ben Wireman <bwireman@ltvco.com>"
COPY --from=build /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=build /go/src/github.com/wiremanb/slack-sr-example/bin/slack-sr-example /usr/local/bin/slack-sr-example
ENTRYPOINT ["/usr/local/bin/slack-sr-example"]
CMD ["--help"]
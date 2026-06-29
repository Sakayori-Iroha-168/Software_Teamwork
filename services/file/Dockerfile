FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/file-service ./cmd/server

FROM alpine:3.22

RUN addgroup -S file && adduser -S -G file file && mkdir -p /data/file-storage && chown -R file:file /data
COPY --from=build /out/file-service /usr/local/bin/file-service

USER file
EXPOSE 8082
ENTRYPOINT ["file-service"]

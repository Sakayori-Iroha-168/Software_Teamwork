FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/knowledge-service ./cmd/server

FROM alpine:3.22

RUN addgroup -S knowledge && adduser -S -G knowledge knowledge
COPY --from=build /out/knowledge-service /usr/local/bin/knowledge-service

USER knowledge
EXPOSE 8083
ENTRYPOINT ["knowledge-service"]

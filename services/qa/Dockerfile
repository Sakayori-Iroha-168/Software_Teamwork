FROM golang:1.26-alpine AS build

WORKDIR /src
COPY services/qa/go.mod ./
RUN go mod download
COPY services/qa/ ./
RUN go build -o /out/qa-service ./cmd/server

FROM alpine:3.22

WORKDIR /app
COPY --from=build /out/qa-service /app/qa-service
EXPOSE 8080
ENTRYPOINT ["/app/qa-service"]

FROM golang:1.22-alpine AS build
WORKDIR /src
COPY services/qa/go.mod ./
COPY services/qa/ ./
RUN go build -o /out/qa ./cmd/server

FROM alpine:3.20
COPY --from=build /out/qa /usr/local/bin/qa
EXPOSE 8080
ENTRYPOINT ["qa"]

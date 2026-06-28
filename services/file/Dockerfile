FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/file-service ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/file-service /file-service
USER nonroot:nonroot
EXPOSE 8082
ENTRYPOINT ["/file-service"]

FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY . ./
RUN go build -buildvcs=false -o /out/qa-server ./cmd/server

FROM alpine:3.20

WORKDIR /app
COPY --from=build /out/qa-server /app/qa-server
EXPOSE 8085
ENTRYPOINT ["/app/qa-server"]

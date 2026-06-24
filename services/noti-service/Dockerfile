# go.sum is not committed (no local Go toolchain); generate it inside the build.
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -trimpath -o /noti-service ./cmd/server

FROM alpine:3.20
RUN adduser -D -u 10001 appuser
COPY --from=build /noti-service /noti-service
USER appuser
EXPOSE 8003
ENTRYPOINT ["/noti-service"]

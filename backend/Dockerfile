FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/main .

FROM alpine:3.22
RUN addgroup -g 10001 -S app && adduser -u 10001 -S -G app app
COPY --from=build --chown=app:app /out/main /app/main
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/app/main"]

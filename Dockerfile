FROM golang:1.24-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server && \
    CGO_ENABLED=0 go build -o /out/consumer ./cmd/consumer

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app/cmd/server

COPY --from=build /out/server ./server
COPY --from=build /out/consumer /app/cmd/consumer/consumer
COPY environment /app/environment
COPY migrations /app/migrations
COPY web /app/web

EXPOSE 8080

CMD ["./server"]

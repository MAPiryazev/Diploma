FROM golang:1.24-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server && \
    CGO_ENABLED=0 go build -o /out/analytics ./cmd/analytics && \
    CGO_ENABLED=0 go build -o /out/relay ./cmd/relay && \
    CGO_ENABLED=0 go build -o /out/projectionbuilder ./cmd/projectionbuilder && \
    CGO_ENABLED=0 go build -o /out/riskevaluation ./cmd/riskevaluation && \
    CGO_ENABLED=0 go build -o /out/transactionexecutor ./cmd/transactionexecutor && \
    CGO_ENABLED=0 go build -o /out/dlqreplay ./cmd/dlqreplay

FROM alpine:3.20 AS runtime-base

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /out /out
COPY config.yaml /app/config.yaml
COPY environment /app/environment
COPY migrations /app/migrations
COPY web /app/web

FROM runtime-base AS server-runtime

WORKDIR /app/cmd/server

COPY --from=runtime-base /out/server ./server
COPY --from=runtime-base /app /app

EXPOSE 8080

CMD ["./server"]

FROM runtime-base AS analytics-runtime

WORKDIR /app/cmd/analytics

COPY --from=runtime-base /out/analytics ./analytics
COPY --from=runtime-base /app /app

EXPOSE 8082

CMD ["./analytics"]

FROM runtime-base AS relay-runtime

WORKDIR /app/cmd/relay

COPY --from=runtime-base /out/relay ./relay
COPY --from=runtime-base /app /app

EXPOSE 8083

CMD ["./relay"]

FROM runtime-base AS projectionbuilder-runtime

WORKDIR /app/cmd/projectionbuilder

COPY --from=runtime-base /out/projectionbuilder ./projectionbuilder
COPY --from=runtime-base /app /app

EXPOSE 2112

CMD ["./projectionbuilder"]

FROM runtime-base AS riskevaluation-runtime

WORKDIR /app/cmd/riskevaluation

COPY --from=runtime-base /out/riskevaluation ./riskevaluation
COPY --from=runtime-base /app /app

EXPOSE 2113

CMD ["./riskevaluation"]

FROM runtime-base AS transactionexecutor-runtime

WORKDIR /app/cmd/transactionexecutor

COPY --from=runtime-base /out/transactionexecutor ./transactionexecutor
COPY --from=runtime-base /app /app

EXPOSE 2114

CMD ["./transactionexecutor"]

FROM runtime-base AS dlqreplay-runtime

WORKDIR /app/cmd/dlqreplay

COPY --from=runtime-base /out/dlqreplay ./dlqreplay
COPY --from=runtime-base /app /app

CMD ["./dlqreplay"]

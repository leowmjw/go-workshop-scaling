FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /part2srv ./cmd/part2srv

FROM scratch
COPY --from=builder /part2srv /part2srv
EXPOSE 8080
ENTRYPOINT ["/part2srv"]

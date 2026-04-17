FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /part1srv ./cmd/part1srv

FROM scratch
COPY --from=builder /part1srv /part1srv
EXPOSE 8080
ENTRYPOINT ["/part1srv"]

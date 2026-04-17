FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /part3srv ./cmd/part3srv

FROM scratch
COPY --from=builder /part3srv /part3srv
EXPOSE 8080
ENTRYPOINT ["/part3srv"]

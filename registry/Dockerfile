FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /mint-registry ./registry

FROM gcr.io/distroless/static-debian12

COPY --from=builder /mint-registry /usr/local/bin/mint-registry

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["mint-registry"]

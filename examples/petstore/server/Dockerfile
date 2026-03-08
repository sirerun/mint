FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /petstore .

FROM gcr.io/distroless/static-debian12

COPY --from=builder /petstore /usr/local/bin/petstore

USER nonroot:nonroot

ENTRYPOINT ["petstore"]
CMD ["--transport", "stdio"]

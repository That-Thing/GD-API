# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

 
COPY . .
 
RUN CGO_ENABLED=0 GOOS=linux go build -o gundeals-api .

FROM alpine:latest

WORKDIR /app
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/gundeals-api /app/

EXPOSE 8080


ENTRYPOINT ["/app/gundeals-api", "-host", "0.0.0.0", "-port", "8080"]
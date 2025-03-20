FROM golang:1.24.1-alpine as builder

RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o main .


FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /app/main ./

EXPOSE 8080

CMD ["./main"]
FROM golang:1.23-alpine

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

COPY .env.example .env

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

EXPOSE 3306
EXPOSE 6379

CMD ["./main"]

FROM golang:1.23.3-alpine

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o /app/server ./cmd/api

EXPOSE 8080

CMD ["./server"]
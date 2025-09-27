FROM golang:1.21-alpine
WORKDIR /app
COPY main.go go.mod go.sum .
COPY password /run/secrets/db-password
RUN go build -o hello-go main.go

CMD ["./hello-go"]
FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /devops-guru ./examples/aws/devops-guru/devops-guru
ENTRYPOINT ["/devops-guru"]

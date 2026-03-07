FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /devops-guru ./examples/aws/devops-guru
ENTRYPOINT ["/devops-guru"]

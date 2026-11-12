FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /devops-guru ./examples/aws/devops-guru
ENTRYPOINT ["/devops-guru"]

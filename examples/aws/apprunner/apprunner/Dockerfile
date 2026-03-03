FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /apprunner ./examples/aws/apprunner/apprunner
ENTRYPOINT ["/apprunner"]

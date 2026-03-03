FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /apprunner-advanced ./examples/aws/apprunner/apprunner-advanced
ENTRYPOINT ["/apprunner-advanced"]

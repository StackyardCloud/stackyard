FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /apprunner-advanced ./examples/apprunner/apprunner-advanced
ENTRYPOINT ["/apprunner-advanced"]

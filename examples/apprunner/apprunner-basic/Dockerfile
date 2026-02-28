FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /apprunner-basic ./examples/apprunner/apprunner-basic
ENTRYPOINT ["/apprunner-basic"]

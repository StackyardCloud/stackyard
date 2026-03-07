FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /application-discovery ./examples/aws/application-discovery
ENTRYPOINT ["/application-discovery"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /application-discovery ./examples/aws/application-discovery/application-discovery
ENTRYPOINT ["/application-discovery"]

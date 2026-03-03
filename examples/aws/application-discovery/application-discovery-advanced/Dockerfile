FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /application-discovery-advanced ./examples/aws/application-discovery/application-discovery-advanced
ENTRYPOINT ["/application-discovery-advanced"]

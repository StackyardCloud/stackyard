FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-portal ./examples/aws/singlesignon-portal
ENTRYPOINT ["/singlesignon-portal"]

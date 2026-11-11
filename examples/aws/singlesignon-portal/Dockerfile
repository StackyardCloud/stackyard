FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-portal ./examples/aws/singlesignon-portal
ENTRYPOINT ["/singlesignon-portal"]

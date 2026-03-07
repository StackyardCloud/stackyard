FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /directoryservice ./examples/aws/directoryservice
ENTRYPOINT ["/directoryservice"]

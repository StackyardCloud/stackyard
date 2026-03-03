FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /directoryservice ./examples/aws/directoryservice/directoryservice
ENTRYPOINT ["/directoryservice"]

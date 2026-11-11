FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /directoryservice ./examples/aws/directoryservice
ENTRYPOINT ["/directoryservice"]

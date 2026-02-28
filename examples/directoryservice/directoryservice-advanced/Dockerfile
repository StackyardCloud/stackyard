FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /directoryservice-advanced ./examples/directoryservice/directoryservice-advanced
ENTRYPOINT ["/directoryservice-advanced"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /directoryservice-basic ./examples/directoryservice/directoryservice-basic
ENTRYPOINT ["/directoryservice-basic"]

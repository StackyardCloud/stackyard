FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /fsx-basic ./examples/fsx/fsx-basic
ENTRYPOINT ["/fsx-basic"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /batch-basic ./examples/batch/batch-basic
ENTRYPOINT ["/batch-basic"]

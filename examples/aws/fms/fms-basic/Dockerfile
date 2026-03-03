FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /fms-basic ./examples/aws/fms/fms-basic
ENTRYPOINT ["/fms-basic"]

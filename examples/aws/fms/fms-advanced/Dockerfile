FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /fms-advanced ./examples/aws/fms/fms-advanced
ENTRYPOINT ["/fms-advanced"]

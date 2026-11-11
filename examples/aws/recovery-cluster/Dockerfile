FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /recovery-cluster ./examples/aws/recovery-cluster
ENTRYPOINT ["/recovery-cluster"]

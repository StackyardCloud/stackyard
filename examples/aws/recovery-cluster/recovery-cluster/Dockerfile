FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /recovery-cluster ./examples/aws/recovery-cluster/recovery-cluster
ENTRYPOINT ["/recovery-cluster"]

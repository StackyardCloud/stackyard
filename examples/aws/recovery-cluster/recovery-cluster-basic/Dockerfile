FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /recovery-cluster-basic ./examples/aws/recovery-cluster/recovery-cluster-basic
ENTRYPOINT ["/recovery-cluster-basic"]

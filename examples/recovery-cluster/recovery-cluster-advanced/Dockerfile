FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /recovery-cluster-advanced ./examples/recovery-cluster/recovery-cluster-advanced
ENTRYPOINT ["/recovery-cluster-advanced"]

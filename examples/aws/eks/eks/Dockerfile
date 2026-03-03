FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /eks ./examples/aws/eks/eks
ENTRYPOINT ["/eks"]

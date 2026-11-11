FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /eks ./examples/aws/eks
ENTRYPOINT ["/eks"]

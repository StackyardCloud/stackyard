FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /omics ./examples/aws/omics
ENTRYPOINT ["/omics"]

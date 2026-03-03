FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /omics ./examples/aws/omics/omics
ENTRYPOINT ["/omics"]

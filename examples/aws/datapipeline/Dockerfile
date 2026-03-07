FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /datapipeline ./examples/aws/datapipeline
ENTRYPOINT ["/datapipeline"]

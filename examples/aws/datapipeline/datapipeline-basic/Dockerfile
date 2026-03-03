FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /datapipeline-basic ./examples/aws/datapipeline/datapipeline-basic
ENTRYPOINT ["/datapipeline-basic"]

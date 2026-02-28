FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /datapipeline-advanced ./examples/datapipeline/datapipeline-advanced
ENTRYPOINT ["/datapipeline-advanced"]

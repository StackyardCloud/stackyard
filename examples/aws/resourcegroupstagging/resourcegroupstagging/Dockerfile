FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /resourcegroupstagging ./examples/aws/resourcegroupstagging/resourcegroupstagging
ENTRYPOINT ["/resourcegroupstagging"]

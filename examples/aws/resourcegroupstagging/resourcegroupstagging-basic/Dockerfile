FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /resourcegroupstagging-basic ./examples/aws/resourcegroupstagging/resourcegroupstagging-basic
ENTRYPOINT ["/resourcegroupstagging-basic"]

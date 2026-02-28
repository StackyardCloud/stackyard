FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /resourcegroupstagging-basic ./examples/resourcegroupstagging/resourcegroupstagging-basic
ENTRYPOINT ["/resourcegroupstagging-basic"]

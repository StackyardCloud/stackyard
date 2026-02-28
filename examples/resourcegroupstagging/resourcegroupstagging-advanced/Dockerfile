FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /resourcegroupstagging-advanced ./examples/resourcegroupstagging/resourcegroupstagging-advanced
ENTRYPOINT ["/resourcegroupstagging-advanced"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudtrail ./examples/aws/cloudtrail/cloudtrail
ENTRYPOINT ["/cloudtrail"]

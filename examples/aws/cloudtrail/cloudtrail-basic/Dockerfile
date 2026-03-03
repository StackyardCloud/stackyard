FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudtrail-basic ./examples/aws/cloudtrail/cloudtrail-basic
ENTRYPOINT ["/cloudtrail-basic"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudtrail-advanced ./examples/aws/cloudtrail/cloudtrail-advanced
ENTRYPOINT ["/cloudtrail-advanced"]

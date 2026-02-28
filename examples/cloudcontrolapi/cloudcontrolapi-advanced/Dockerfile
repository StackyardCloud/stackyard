FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudcontrolapi-advanced ./examples/cloudcontrolapi/cloudcontrolapi-advanced
ENTRYPOINT ["/cloudcontrolapi-advanced"]

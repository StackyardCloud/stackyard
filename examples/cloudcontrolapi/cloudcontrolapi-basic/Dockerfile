FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudcontrolapi-basic ./examples/cloudcontrolapi/cloudcontrolapi-basic
ENTRYPOINT ["/cloudcontrolapi-basic"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /systems-manager-basic ./examples/aws/systems-manager/systems-manager-basic
ENTRYPOINT ["/systems-manager-basic"]

FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudhsm ./examples/aws/cloudhsm
ENTRYPOINT ["/cloudhsm"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudhsm ./examples/aws/cloudhsm/cloudhsm
ENTRYPOINT ["/cloudhsm"]

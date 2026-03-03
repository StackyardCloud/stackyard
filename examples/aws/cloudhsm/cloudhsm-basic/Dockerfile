FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudhsm-basic ./examples/aws/cloudhsm/cloudhsm-basic
ENTRYPOINT ["/cloudhsm-basic"]

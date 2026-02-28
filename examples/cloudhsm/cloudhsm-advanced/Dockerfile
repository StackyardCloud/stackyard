FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudhsm-advanced ./examples/cloudhsm/cloudhsm-advanced
ENTRYPOINT ["/cloudhsm-advanced"]

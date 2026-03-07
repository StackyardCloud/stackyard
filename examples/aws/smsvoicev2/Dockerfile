FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /smsvoicev2 ./examples/aws/smsvoicev2
ENTRYPOINT ["/smsvoicev2"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /smsvoicev2 ./examples/aws/smsvoicev2/smsvoicev2
ENTRYPOINT ["/smsvoicev2"]

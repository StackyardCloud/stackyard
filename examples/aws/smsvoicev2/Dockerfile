FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /smsvoicev2 ./examples/aws/smsvoicev2
ENTRYPOINT ["/smsvoicev2"]

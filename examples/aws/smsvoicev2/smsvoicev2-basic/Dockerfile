FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /smsvoicev2-basic ./examples/aws/smsvoicev2/smsvoicev2-basic
ENTRYPOINT ["/smsvoicev2-basic"]

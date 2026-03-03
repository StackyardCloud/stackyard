FROM golang:1.23-alpine

WORKDIR /app
COPY . .
RUN go build -o /partner-central-basic ./examples/aws/partner-central/partner-central-basic

CMD ["/partner-central-basic"]

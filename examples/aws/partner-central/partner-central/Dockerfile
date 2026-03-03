FROM golang:1.23-alpine

WORKDIR /app
COPY . .
RUN go build -o /partner-central ./examples/aws/partner-central/partner-central

CMD ["/partner-central"]

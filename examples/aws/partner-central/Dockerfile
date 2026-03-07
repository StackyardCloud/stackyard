FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /partner-central ./examples/aws/partner-central

CMD ["/partner-central"]

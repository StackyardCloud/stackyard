FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /partner-central ./examples/aws/partner-central

CMD ["/partner-central"]

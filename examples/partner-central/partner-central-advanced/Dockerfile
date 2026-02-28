FROM golang:1.23-alpine

WORKDIR /app
COPY . .
RUN go build -o /partner-central-advanced ./examples/partner-central/partner-central-advanced

CMD ["/partner-central-advanced"]

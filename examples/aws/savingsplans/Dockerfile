FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /savingsplans ./examples/aws/savingsplans
ENTRYPOINT ["/savingsplans"]

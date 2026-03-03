FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /savingsplans ./examples/aws/savingsplans/savingsplans
ENTRYPOINT ["/savingsplans"]

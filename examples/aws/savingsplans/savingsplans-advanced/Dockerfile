FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /savingsplans-advanced ./examples/aws/savingsplans/savingsplans-advanced
ENTRYPOINT ["/savingsplans-advanced"]

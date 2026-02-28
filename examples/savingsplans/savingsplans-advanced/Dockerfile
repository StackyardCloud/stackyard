FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /savingsplans-advanced ./examples/savingsplans/savingsplans-advanced
ENTRYPOINT ["/savingsplans-advanced"]

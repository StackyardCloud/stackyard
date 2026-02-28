FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /savingsplans-basic ./examples/savingsplans/savingsplans-basic
ENTRYPOINT ["/savingsplans-basic"]

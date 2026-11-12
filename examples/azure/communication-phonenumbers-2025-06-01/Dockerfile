FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /communication-phonenumbers-2025-06-01 ./examples/azure/communication-phonenumbers-2025-06-01

ENTRYPOINT ["/communication-phonenumbers-2025-06-01"]

FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /communication-siprouting-2023-03-01 ./examples/azure/communication-siprouting-2023-03-01

ENTRYPOINT ["/communication-siprouting-2023-03-01"]

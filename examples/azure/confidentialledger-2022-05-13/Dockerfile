FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /confidentialledger-2022-05-13 ./examples/azure/confidentialledger-2022-05-13

ENTRYPOINT ["/confidentialledger-2022-05-13"]

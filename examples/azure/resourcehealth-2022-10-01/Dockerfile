FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /resourcehealth-2022-10-01 ./examples/azure/resourcehealth-2022-10-01

ENTRYPOINT ["/resourcehealth-2022-10-01"]

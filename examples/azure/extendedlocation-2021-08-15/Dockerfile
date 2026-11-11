FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /extendedlocation-2021-08-15-example ./examples/azure/extendedlocation-2021-08-15

ENTRYPOINT ["/extendedlocation-2021-08-15-example"]

FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /customerlockbox-2018-02-28-preview ./examples/azure/customerlockbox-2018-02-28-preview

ENTRYPOINT ["/customerlockbox-2018-02-28-preview"]

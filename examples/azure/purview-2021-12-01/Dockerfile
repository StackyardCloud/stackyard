FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /purview-2021-12-01 ./examples/azure/purview-2021-12-01

ENTRYPOINT ["/purview-2021-12-01"]

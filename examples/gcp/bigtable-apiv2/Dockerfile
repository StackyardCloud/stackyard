FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /bigtable-apiv2 ./examples/gcp/bigtable-apiv2
CMD ["/bigtable-apiv2"]

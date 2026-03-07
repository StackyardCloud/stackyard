FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /bigtable-apiv2 ./examples/gcp/bigtable-apiv2
CMD ["/bigtable-apiv2"]

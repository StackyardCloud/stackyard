FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /pcs-basic ./examples/pcs/pcs-basic
ENTRYPOINT ["/pcs-basic"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /pcs-advanced ./examples/pcs/pcs-advanced
ENTRYPOINT ["/pcs-advanced"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /omics-advanced ./examples/omics/omics-advanced
ENTRYPOINT ["/omics-advanced"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /omics-basic ./examples/omics/omics-basic
ENTRYPOINT ["/omics-basic"]

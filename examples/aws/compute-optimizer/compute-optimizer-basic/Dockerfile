FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /compute-optimizer-basic ./examples/aws/compute-optimizer/compute-optimizer-basic
ENTRYPOINT ["/compute-optimizer-basic"]

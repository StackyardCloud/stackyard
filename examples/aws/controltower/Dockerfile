FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /controltower ./examples/aws/controltower
ENTRYPOINT ["/controltower"]

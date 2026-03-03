FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /controltower ./examples/aws/controltower/controltower
ENTRYPOINT ["/controltower"]

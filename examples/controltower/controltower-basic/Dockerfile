FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /controltower-basic ./examples/controltower/controltower-basic
ENTRYPOINT ["/controltower-basic"]

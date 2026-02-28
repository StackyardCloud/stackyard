FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /controltower-advanced ./examples/controltower/controltower-advanced
ENTRYPOINT ["/controltower-advanced"]

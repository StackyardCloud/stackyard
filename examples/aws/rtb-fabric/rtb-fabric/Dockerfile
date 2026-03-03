FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /rtb-fabric ./examples/aws/rtb-fabric/rtb-fabric
ENTRYPOINT ["/rtb-fabric"]

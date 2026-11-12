FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /rtb-fabric ./examples/aws/rtb-fabric
ENTRYPOINT ["/rtb-fabric"]

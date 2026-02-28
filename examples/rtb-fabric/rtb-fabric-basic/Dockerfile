FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /rtb-fabric-basic ./examples/rtb-fabric/rtb-fabric-basic
ENTRYPOINT ["/rtb-fabric-basic"]

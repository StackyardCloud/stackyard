FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /rtb-fabric-advanced ./examples/rtb-fabric/rtb-fabric-advanced
ENTRYPOINT ["/rtb-fabric-advanced"]

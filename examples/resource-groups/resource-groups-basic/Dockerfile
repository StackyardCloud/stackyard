FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /resource-groups-basic ./examples/resource-groups/resource-groups-basic
ENTRYPOINT ["/resource-groups-basic"]

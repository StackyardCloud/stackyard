FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /resource-groups-advanced ./examples/resource-groups/resource-groups-advanced
ENTRYPOINT ["/resource-groups-advanced"]

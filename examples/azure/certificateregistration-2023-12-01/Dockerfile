FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /certificateregistration-2023-12-01 ./examples/azure/certificateregistration-2023-12-01

ENTRYPOINT ["/certificateregistration-2023-12-01"]

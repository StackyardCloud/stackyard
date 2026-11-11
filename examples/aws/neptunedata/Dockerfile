FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptunedata ./examples/aws/neptunedata
ENTRYPOINT ["/neptunedata"]

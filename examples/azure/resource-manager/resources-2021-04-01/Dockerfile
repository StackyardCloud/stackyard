FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/resources-2021-04-01 ./examples/azure/resource-manager/resources-2021-04-01

FROM gcr.io/distroless/base-debian12

COPY --from=builder /out/resources-2021-04-01 /resources-2021-04-01

ENTRYPOINT ["/resources-2021-04-01"]

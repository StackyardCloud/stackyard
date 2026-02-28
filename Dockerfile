FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/stackyard ./cmd/stackyard

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/stackyard /stackyard

EXPOSE 4566
ENV STACKYARD_ADDR=:4566

ENTRYPOINT ["/stackyard"]

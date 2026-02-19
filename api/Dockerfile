FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /aegis-api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /aegis-api /aegis-api
EXPOSE 8080
ENTRYPOINT ["/aegis-api"]

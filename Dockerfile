FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /fatec-turbo .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /fatec-turbo /fatec-turbo
ENV FT_ADDR=0.0.0.0:8027 TZ=America/Sao_Paulo
EXPOSE 8027
USER nonroot
ENTRYPOINT ["/fatec-turbo"]

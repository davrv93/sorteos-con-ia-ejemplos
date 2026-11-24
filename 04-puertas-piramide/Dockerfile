# Se construye dentro de Docker: no necesitas Go en tu computadora.
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /sorteo .

FROM alpine:3.22
RUN mkdir /datos && chown 65534:65534 /datos
COPY --from=build /sorteo /sorteo
USER 65534:65534
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/sorteo"]

FROM golang:1.25.4-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server .
FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D app && mkdir /data && chown app /data
USER app
COPY --from=build /server /server
ENV PORT=8080 DATABASE_PATH=/data/geofence.db
EXPOSE 8080
ENTRYPOINT ["/server"]

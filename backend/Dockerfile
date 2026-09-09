FROM golang:1.22-alpine AS build
WORKDIR /src

# Cache module downloads separately from source changes
COPY backend/go.mod backend/go.sum* ./backend/
RUN cd backend && go mod download

COPY backend ./backend
RUN cd backend && CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/server ./server

# The backend reads ward boundaries / demographics fixtures relative to a
# "repo root" it detects at runtime (HEEWS_ROOT). Bake the data/ folder into
# the image and point it there explicitly so real ward polygons load instead
# of falling back to 30 synthetic same-coordinate wards.
COPY data ./data
ENV HEEWS_ROOT=/app

EXPOSE 8080
ENTRYPOINT ["./server"]
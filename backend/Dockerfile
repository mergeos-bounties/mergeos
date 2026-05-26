FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/mergeos ./cmd/mergeos

FROM alpine:3.22

RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /app

ENV PORT=8080 \
    MERGEOS_ENV=local \
    DATABASE_URL=postgres://mergeos:mergeos@postgres:5432/mergeos_local?sslmode=disable \
    MERGEOS_STATE_PATH=/data/mergeos-state.local.json \
    TOKEN_SYMBOL=MRG \
    PLATFORM_FEE_BPS=1000 \
    DEV_PAYMENT_ENABLED=true \
    DEV_PAYMENT_CODE=LOCAL-PAID \
    ADMIN_AUTO_PROMOTE_FIRST_USER=true \
    ADMIN_EMAIL=admin@gmail.com \
    ADMIN_PASSWORD=Admin123 \
    ADMIN_NAME="MergeOS Admin" \
    ADMIN_COMPANY_NAME=MergeOS \
    GITHUB_OAUTH_CLIENT_ID="" \
    GITHUB_OAUTH_CLIENT_SECRET="" \
    BOUNTY_ROOT=/data/bounties \
    UPLOAD_ROOT=/data/uploads \
    SSL_REVIEW_ENABLED=false \
    SMTP_FROM=noreply@mergeos.local

COPY --from=build /out/mergeos /app/mergeos

EXPOSE 8080

CMD ["/app/mergeos"]

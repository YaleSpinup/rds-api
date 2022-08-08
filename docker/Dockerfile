# This is a multi-stage Dockerfile and requires >= Docker 17.05
# https://docs.docker.com/engine/userguide/eng-image/multistage-build/
FROM gobuffalo/buffalo:v0.18.8 as builder

ARG version=0.0.0
ARG prerelease
ARG githash=""
ARG buildstamp=""

RUN mkdir /app
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
ADD . .
RUN go version
RUN buffalo build --static -o /bin/api --ldflags "-X github.com/YaleSpinup/rds-api/rdsapi.Version=$version -X github.com/YaleSpinup/rds-api/rdsapi.VersionPrerelease=$prerelease -X github.com/YaleSpinup/rds-api/rdsapi.GitHash=$githash -X github.com/YaleSpinup/rds-api/rdsapi.BuildStamp=$buildstamp"


FROM python:3-alpine
RUN apk add --no-cache bash ca-certificates && \
    pip3 install awscli --upgrade && \
    aws --version

WORKDIR /app/

COPY --from=builder /bin/api ./
COPY docker/import_config.sh ./

ENV GO_ENV=production
ENV PORT=8080
ENV ADDR=0.0.0.0
EXPOSE 8080

RUN chown -R nobody:nogroup .
USER nobody

CMD ./import_config.sh && ./api

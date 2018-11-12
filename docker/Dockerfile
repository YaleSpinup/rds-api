# This is a multi-stage Dockerfile and requires >= Docker 17.05
# https://docs.docker.com/engine/userguide/eng-image/multistage-build/
FROM gobuffalo/buffalo:v0.13.5 as builder

RUN mkdir -p $GOPATH/src/github.com/YaleSpinup/rds-api
WORKDIR $GOPATH/src/github.com/YaleSpinup/rds-api

ADD . .
RUN go version
RUN dep ensure
RUN buffalo build --static -o /bin/api

FROM alpine
RUN apk add --no-cache bash
RUN apk add --no-cache ca-certificates
RUN apk add --no-cache wget python gettext && \
    wget -nv "https://s3.amazonaws.com/aws-cli/awscli-bundle.zip" && \
    unzip awscli-bundle.zip && \
    ./awscli-bundle/install -i /usr/local/aws -b /usr/local/bin/aws && \
    /usr/local/bin/aws --version

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

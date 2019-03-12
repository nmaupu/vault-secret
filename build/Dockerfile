FROM alpine:3.8

RUN apk update && \
    apk add ca-certificates curl jq bash && \
    rm -rf /var/cache/apk/*

ENV OPERATOR=/usr/local/bin/vault-secret \
    USER_UID=1001 \
    USER_NAME=vault-secret

# install operator binary
COPY build/_output/bin/vault-secret ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}

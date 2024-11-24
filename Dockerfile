FROM alpine:3.20
WORKDIR /
COPY renovator renovator
USER nobody
ENTRYPOINT ["/renovator"]

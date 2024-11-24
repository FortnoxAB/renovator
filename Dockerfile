FROM gcr.io/distroless/static-debian12:nonroot
COPY renovator /renovator
USER nonroot
ENTRYPOINT ["/renovator"]

FROM scratch

COPY consul-connect-router /
ENTRYPOINT ["/consul-connect-router"]

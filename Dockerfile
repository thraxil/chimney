FROM centurylink/ca-certs
COPY chimney /
ENTRYPOINT ["/chimney", "-config=/config.json"]

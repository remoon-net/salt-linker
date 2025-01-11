FROM alpine:latest
RUN   apk add --no-cache  ca-certificates
WORKDIR /app
VOLUME [ "/app/pb_data/" ]
EXPOSE 8090

COPY salt-linker /app/salt-linker
# start PocketBase
ENTRYPOINT [ "/app/salt-linker", "serve", "--http=0.0.0.0:8090" ]
CMD []

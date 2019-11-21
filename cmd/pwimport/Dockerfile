FROM alpine:latest
RUN apk add --no-cache bash
ADD pwimport /
ADD docker-entrypoint.sh /
CMD ["/docker-entrypoint.sh"]

# Runtime image for local/up.sh, packs artifacts built by local/build.sh.
# The regular Dockerfile in the repo root builds everything in one go.
FROM scratch
COPY neko_rooms /app/bin/neko_rooms
COPY www/ /var/www

ENV NEKO_ROOMS_BIND=:8080
ENV NEKO_ROOMS_ADMIN_STATIC=/var/www
ENV NEKO_ROOMS_DATA_DIR=/data
VOLUME /data

EXPOSE 8080

ENTRYPOINT [ "/app/bin/neko_rooms" ]
CMD [ "serve" ]

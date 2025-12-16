FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7-1764794109

WORKDIR /app

ADD . .

ENV CERT_PATH "/usr/uipath"

RUN chgrp -R 0 /app && \
    chmod -R g=u /app

USER 1000:1000

CMD [ "/app/update-cert-trust.sh" ]
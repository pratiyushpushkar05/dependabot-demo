FROM debian:stable

RUN apt-get update 
RUN apt-get upgrade -y
RUN apt-get install -y ca-certificates

WORKDIR /app

ADD . .

ENV CERT_PATH "/usr/uipath"

RUN chmod +x "/app/update-cert-trust.sh"

RUN chgrp -R 0 /app && \
    chmod -R g=u /app

USER 1000:1000

CMD [ "/app/update-cert-trust.sh" ]
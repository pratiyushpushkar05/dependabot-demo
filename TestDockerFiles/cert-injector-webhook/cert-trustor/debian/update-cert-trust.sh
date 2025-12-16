#!/bin/bash

certificate_path="${CERT_PATH}"

if [[ -z "${certificate_path}" ]]; then
    echo "Certificate not found"
    exit
fi

echo "coping ${certificate_path}/* to /usr/local/share/ca-certificates/"
cp "${certificate_path}"/* "/usr/local/share/ca-certificates/"

counter=0
for file in /usr/local/share/ca-certificates/*.crt; do     
  if [[ -f "$file" ]]; then
    OLDIFS=$IFS; IFS=';' blocks=$(sed -n '/-----BEGIN /,/-----END/ {/-----BEGIN / s/^/\;/; p}'  "$file");
    for block in ${blocks#;}; do 
        #shellcheck disable=SC2086
        echo $block > "/usr/local/share/ca-certificates/cert-${counter}.crt"
        counter=$(( counter + 1 ))
    done; IFS=$OLDIFS 
  fi
done


echo "updating certificate trust store"
echo "update-ca-certificates"
update-ca-certificates

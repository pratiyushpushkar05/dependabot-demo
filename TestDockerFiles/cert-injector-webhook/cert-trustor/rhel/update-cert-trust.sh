#!/bin/bash

certificate_path="${CERT_PATH}"

if [[ -z "${certificate_path}" ]]; then
    echo "Certificate not found"
    exit
fi

echo "coping ${certificate_path}/* to /etc/pki/ca-trust/source/anchors/"
cp "${certificate_path}"/* "/etc/pki/ca-trust/source/anchors/"

echo "updating certificate trust store"
echo "update-ca-trust extract"
update-ca-trust extract

echo "trust list"
trust list

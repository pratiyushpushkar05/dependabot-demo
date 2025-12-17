# Builder stage for cmctl with updated containerd dependency
FROM mzwkossalpweacr.azurecr.io/echo/docker.io/library/golang:1-20251119 AS cmctl-builder

ARG CMCTL_VERSION=v2.3.0

RUN apt-get update && \
    apt-get install -y git && \
    rm -rf /var/lib/apt/lists/* && \
    git clone --filter=tree:0 https://github.com/cert-manager/cmctl.git /build/cmctl && \
    cd /build/cmctl && \
    git checkout tags/${CMCTL_VERSION} && \
    # Update containerd dependency from v1.7.27 to v1.7.29
    go get github.com/containerd/containerd@v1.7.29 && \
    go mod tidy && \
    # Build cmctl binary
    GOARCH=amd64 GOOS=linux go build -o cmctl .

# Final stage
FROM registry.access.redhat.com/ubi9/ubi:9.7-1764794285

LABEL maintainer="Service Fabric Core Team"
LABEL org.opencontainers.builder.image.vendor="echohq.com"

ENV TZ=America/New_York
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# Copy Python requirements file
COPY requirements.txt /tmp/requirements.txt

RUN dnf install https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm -y && \
    dnf install http://rpms.remirepo.net/enterprise/remi-release-9.rpm -y && \
    dnf update -y && \
    dnf install -y wget zip unzip jq git python3 python3-pip bind-utils net-tools openssl bc krb5-workstation krb5-libs netcat iputils iperf && \
    rm -rf /usr/lib/python3.9/site-packages/requests* /usr/lib/python3.9/site-packages/urllib3* /usr/lib/python3.9/site-packages/idna* /usr/lib/python3.9/site-packages/charset_normalizer* /usr/lib/python3.9/site-packages/setuptools* && \
    rm -rf /usr/lib64/python3.9/site-packages/requests* /usr/lib64/python3.9/site-packages/urllib3* /usr/lib64/python3.9/site-packages/idna* /usr/lib64/python3.9/site-packages/charset_normalizer* /usr/lib64/python3.9/site-packages/setuptools* && \
    pip3 install --ignore-installed -r /tmp/requirements.txt && \
    curl https://packages.microsoft.com/config/rhel/9/prod.repo | tee /etc/yum.repos.d/microsoft.repo && \
    dnf install -y powershell && \
    curl -L https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash -s -- --version v4.0.0 && \
    export KUSTOMIZE_VER=v5.8.0 && \
    wget -qO- https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/${KUSTOMIZE_VER}/kustomize_${KUSTOMIZE_VER}_linux_amd64.tar.gz | tar -xvz -C /usr/bin && \
    chmod +x /usr/bin/kustomize && \
    VERSION=$(curl -L -s https://raw.githubusercontent.com/argoproj/argo-cd/stable/VERSION) \
    && curl -sSL -o /usr/local/bin/argocd https://github.com/argoproj/argo-cd/releases/download/v$VERSION/argocd-linux-amd64 && \
    chmod +x /usr/local/bin/argocd && \
    curl -sLf -v https://dl.k8s.io/release/v1.34.1/bin/linux/amd64/kubectl > /usr/bin/kubectl && \
    chmod +x /usr/bin/kubectl && \
    dnf install https://github.com/rclone/rclone/releases/download/v1.71.2/rclone-v1.71.2-linux-amd64.rpm -y

# Copy cmctl binary from builder stage
COPY --from=cmctl-builder /build/cmctl/cmctl /usr/bin/cmctl
RUN chmod +x /usr/bin/cmctl

RUN curl https://packages.microsoft.com/config/rhel/9/prod.repo > /etc/yum.repos.d/mssql-release.repo
RUN yum search odbc && \
    yum search msodbcsql17 && \
    ACCEPT_EULA=Y yum install -y unixODBC unixODBC-devel && \
    yum download -y msodbcsql17 && \
    ACCEPT_EULA=Y rpm -Uvh --nodeps msodbcsql17*rpm && \
    ACCEPT_EULA=Y dnf -y install mssql-tools

# s3cmd fips support fix is not part of s3cmd release. Use dnf to install s3cmd once its release - https://github.com/s3tools/s3cmd/pull/1233
RUN git clone https://github.com/s3tools/s3cmd.git && cd s3cmd &&  python3 setup.py sdist && pip3 install ./dist/s3cmd-2.4.0.dev0.tar.gz

USER 1001
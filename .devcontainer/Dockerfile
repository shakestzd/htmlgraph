# syntax=docker/dockerfile:1.7

FROM mcr.microsoft.com/devcontainers/python:1-3.11-bookworm

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash-completion \
        build-essential \
        curl \
        fd-find \
        jq \
        less \
        make \
        pkg-config \
        ripgrep \
        shellcheck \
        sqlite3 \
        tree \
        unzip \
        xz-utils \
    && rm -rf /var/lib/apt/lists/* \
    && python3 -m pip install --no-cache-dir --upgrade pip uv

RUN if [ -x /usr/bin/fdfind ] && [ ! -e /usr/local/bin/fd ]; then ln -s /usr/bin/fdfind /usr/local/bin/fd; fi

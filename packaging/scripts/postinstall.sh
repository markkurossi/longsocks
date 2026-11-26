#!/bin/sh

set -e

if ! getent group longsocks >/dev/null; then
    addgroup --system longsocks
fi

if ! getent passwd longsocks >/dev/null; then
    adduser --system \
            --ingroup longsocks \
            --no-create-home \
            --home /var/lib/longsocks \
            --shell /usr/sbin/nologin \
            longsocks
fi

install -d -o root -g longsocks -m 0750 /etc/longsocks
install -d -o longsocks -g longsocks -m 0750 /var/lib/longsocks

systemctl daemon-reload || true

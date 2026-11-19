# longsocks

Longsocks is a [SOCKS5](https://www.rfc-editor.org/info/rfc1928/)
reverse proxy that provides access to private machines through
long-lived outbound tunnels.

<p align="center">
  <img src="longsocks.png" width="320">
</p>

``` text
                 ┌──────────────────────────┐
                 │        longsocksd        │
                 │                          │
 SSH ──SOCKS5──► │   SOCKS listener :1080   │
 HTTPS           │            │             │
                 │            ▼             │           ┌──────────┐
                 │    connection manager  ◄─┼──control──│  diald   │
                 │       │    │    │        │           │          │
                 │       │    │    └────────┼───────────┼──proxy───┼──► :22
                 │       │    │             │           │          │
                 │       │    └─────────────┼───────────┼──proxy───┼──► :22
                 │       │                  │           │          │
                 │       └──────────────────┼───────────┼──proxy───┼──► :443
                 │                          │           │          │
 longsocks       │   /tmp/longsocks.sock    │           └──────────┘
     │           └──────────────────────────┘
     │                        ▲
     │                        │
     └──────────ipc───────────┘
```

**longsocksd**
The server-side relay and SOCKS5 proxy.

**diald**
The daemon running on target machines that establishes outbound
tunnels.

**longsocks**
The proxy client command.

## Setup

Run `longsocksd` on an Internet-reachable server. This is the
rendezvous server that connects clients with the servers.

``` shell
/usr/local/sbin/longsocksd
```

The rendezvous server also has the `longsocks` command that can be
used to control the `longsocksd` and register new clients.

Install and run `diald` on each target server.

``` shell
/usr/local/sbin/diald
```

## Registering a Server

New hosts are first registered at the rendezvous server with the
`longsocks` command:

``` shell
longsocks -d host-init test.lab.ephemelier.com

Save this configuration file to host at /usr/local/etc/longsocksd/diald.toml:

----------------------------------------------------------------------
# Diald config for [test.lab.ephemelier.com]

[longsocksd]

hostname = "localhost"
port = 8103

[diald]

ca = """
-----BEGIN CERTIFICATE-----
MIICFjCCAb2gAwIBAgIIcprOWLlLaggwCgYIKoZIzj0EAwQwNjELMAkGA1UEBhMC
RkkxEzARBgNVBAoTCkVwaGVtZWxpZXIxEjAQBgNVBAMTCUxvbmdzb2NrczAeFw0y
NjA5MTgwNDQ4MjBaFw00NjA5MTMwNDQ4MjBaMDYxCzAJBgNVBAYTAkZJMRMwEQYD
VQQKEwpFcGhlbWVsaWVyMRIwEAYDVQQDEwlMb25nc29ja3MwWTATBgcqhkjOPQIB
BggqhkjOPQMBBwNCAAR0zQOHrGmJ86IbaN/X7a36gJv5MXp0yWCSh5dbJI6NfaxR
wCZI60ah3E1kx3CHrb9wFEIIJY0jagYeWEMlJIe1o4G0MIGxMA4GA1UdDwEB/wQE
AwIBhjAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwEgYDVR0TAQH/BAgw
BgEB/wIBADAdBgNVHQ4EFgQUTaDJN6rbYzTOp7hVVFe7nX6m7qgwTQYDVR0RBEYw
RIIJbG9jYWxob3N0gglsb2NhbGhvc3SCCyoubG9jYWxob3N0ggcqLmxvY2FshwR/
AAABhxAAAAAAAAAAAAAAAAAAAAABMAoGCCqGSM49BAMEA0cAMEQCIGgyyC1qOWKe
REi4sv2JaAhI43QAfRt3+gsyYIyWse/MAiBCvnNpdbWCZwKFiz0TWksk4F7Xv79m
cv+3LD6s8Ppaig==
-----END CERTIFICATE-----
"""

certificate_file = "/usr/local/etc/longsocks.d/diald.crt"
private_key_file = "/usr/local/etc/longsocks.d/diald.key"
----------------------------------------------------------------------

Run this command at host to init diald:

----------------------------------------------------------------------
diald init 91EE56B7-9EFF-4EC5-AD33-EA61E9339095
----------------------------------------------------------------------
```

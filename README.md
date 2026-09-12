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
     └────────────────────────┘
```

**longsocksd**
The server-side relay and SOCKS5 proxy.

**diald**
The daemon running on target machines that establishes outbound
tunnels.

**longsocks**
The proxy client command.

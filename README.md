# longsocks

Longsocks is a [SOCKS5](https://www.rfc-editor.org/info/rfc1928/)
reverse proxy that provides access to private machines through
long-lived outbound tunnels.

<p align="center">
  <img src="longsocks.png" width="320">
</p>

**longsocksd**
The server-side relay and SOCKS5 proxy.

**diald**
The daemon running on target machines that establishes outbound
tunnels.

**longsock**
The proxy client command.

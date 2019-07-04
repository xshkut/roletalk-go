
<!-- @import "[TOC]" {cmd="toc" depthFrom=1 depthTo=6 orderedList=false} -->
# ROLETALK

<!-- code_chunk_output -->

- [ROLETALK](#roletalk)
  - [Overview](#overview)
  - [Conception](#conception)
  - [Use case](#use-case)

<!-- /code_chunk_output -->

## Overview

Roletalk is microservice peer-to-peer communication framework. Asynchronous and event-driven.
Developed in honor of scalability, simplicity and efficiency.

Roletalk uses Websocket for data transferring. Websocket was chosen for TCP framing with minimal network overhead. In addition it allows internet browsers to work as services if needed.

Currently there is corresponding wire-compatible JS framework.

## Conception

...

## Use case

Choose roletalk if you need to build brokerless architecture. Such approach brings some advantages: minimal RTT (round-trip time), less network transferring, no SPOF (single point of failure). In some scenarios broker can be a bottleneck also.
Roletalk lets you build flexible communication patterns:

- Publish-Subscribe,
- RPC (request-reply),
- Event (single message with optional payload),
- Pipeline (sequental distributed data processing),
- Bus (many to many communication),
- Survey (request to multiple peers).

Disadvantages: all the disadvantages of brokerless architecture take their place here.
Also framework is not actually polyglot. Currently it is implemented in Go and JS (NodeJS)

Features

• Round-robin client-side load balancing between roles declared on remote peers;

• Client as service: no matter who established connection. Both listener and dialer of a connection are services.
It allows you to deploy services not exposing them to Internet or behind private network etc.
Of course such instances should connect to listeners they have network tcp access to.

• No heavy internal message conversions (json/xml serialization/parsing). Just binary to utf-8/int and vice-versa.

• Streams available. Transfer large or unknown amounts of data via streams.

• Optional TLS on transport layer.

• Automatic service-recognition. Just connect services together and each of them will know each about other's roles, ID, name and meta info.
If peer's role gets disabled or enabled each connected peer immediately gets informed and rebuilds its internal state.

• Redunant connections. There could be multiple connections between two peers. All communication is load-balanced between them.
With auto-reconnection this can be useful to keep in pair two peers which are connected to each other in case of one peer's IP address is changed.

• Scalable authentication: each peer can have zero or multiple ID:KEY combinations.

Security

To achieve strong MITM-protection use TLS connection.

Package is provided with simple in-built authentication mechanism: preshared ID:KEY pairs. Peers which are supposed to be connected should have at least one common ID:KEY pair.

The authentication process between two peers is simple:

1. Each peer sends list of his auth ID's and randomly generated CHALLENGE to another. If no auth ID:KEY specified peer sends auth confirmation.

2. Each peer checks if locally registered ID's intersect with ones received from remote peer. Alse sends error and closes connection

3. Each peer chooses random ID from intersection list and hashes received CHALLENGE with corresponding KEY of chosen ID. Sends hashed RESPONSE value with that ID.

4. Each peer receives RESPONSE with ID and checks if CHALLENGE hashed with KEY of received ID equals received RESPONSE. In such case sends auth confirmation. Else rejects auth and closes connection.

5. In case of both peers confirm each others's RESPONSES the handshake process is complete and then connections take part in communication. If auth exceeds timeout or some peer sends error and/or closes connection handshake fails.

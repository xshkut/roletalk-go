# ROLETALK

## Overview

Roletalk is peer-to-peer communication framework for microservices; asynchronous and event-driven.
Developed in honor of scalability, simplicity and efficiency.

Roletalk internally uses Websocket for data transferring. Websocket was chosen as TCP framing tool with minimal network overhead.

Currently there is corresponding wire-compatible JS framework.

## Conception and vocabulary

...

## Use case

Choose roletalk if you need to build brokerless architecture. Such approach brings some advantages: minimal RTT (round-trip time), less network transferring, no SPOF (single point of failure). In some scenarios broker can be a bottleneck also.
Roletalk lets you implement flexible communication patterns:

- Publish-Subscribe,
- RPC (request-reply),
- Event (single message with optional payload),
- Pipeline (sequental distributed data processing),
- Bus (many to many communication),
- Survey (request to multiple peers).

## Features

• Round-robin client-side load balancing between roles (services) declared on remote peers;

• Client as service: no matter who establish connection. Both listener and dialer of a connection are services.
It allows you to deploy services not exposing them to the Internet or behind private network etc.
Of course such instances should connect to listeners which they have network access to.

• No internal heavy message conversions (json/xml serialization/parsing). Just binary to utf-8/int and vice-versa.

• Binary streams. Transfer large or unknown amounts of data via streams.

• Optional TLS on transport layer.

• Automatic service-recognition. Just connect peers together and each of them will know each about other's roles (services), IDs, names and meta info.
If peer's role gets disabled or enabled each connected peer immediately gets informed and rebuilds its internal state.

• Redunant connections support. There could be multiple connections between two peers. All communication is load-balanced between them.
With auto-reconnection this can be useful to keep connected two peers which are in case of one peer's IP address is changed.

• Scalable authentication: each peer can have zero or multiple ID:KEY combinations.

## Security

To achieve strong MITM-protection use TLS connection.

Package is provided with simple in-built authentication mechanism: preshared ID:KEY pairs. Peers which are supposed to be connected should have at least one common ID:KEY pair.

The authentication process between two peers can be described in few steps:

1. Each peer sends a list of his auth ID's and randomly generated CHALLENGE to another. If no auth ID:KEY specified peer sends auth confirmation.

2. Each peer checks if locally registered list of ID's intersects with the one received from remote peer. If no ID matched, sends error and closes conenctions with corresponding code.

3. Each peer chooses random intersected ID and hashes received CHALLENGE with corresponding KEY for that ID. Sends hashed RESPONSE value in pair with that ID.

4. Each peer receives RESPONSE with ID and checks if CHALLENGE hashed with corresponding KEY of received ID equals received RESPONSE. In such case sends auth confirmation. Else rejects auth and closes connection.

5. In case of both peers confirm each others's RESPONSES the handshake process is complete and then connections take part in futher communication. If auth exceeds timeout or some peer sends error and/or closes connection handshake fails.

# ROLETALK

<!-- vscode-markdown-toc -->
* [ Overview](#Overview)
* [ Use case](#Usecase)
* [ Features](#Features)
* [ Concept](#Concept)
	* [ Structure](#Structure)
	* [ Communication](#Communication)
* [ Data types](#Datatypes)
	* [ Acquaintance](#Acquaintance)
* [ Security](#Security)
* [ Contribution](#Contribution)

<!-- vscode-markdown-toc-config
	numbering=false
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

## <a name='Overview'></a> Overview

Roletalk is peer-to-peer communication framework for microservices; asynchronous and event-driven.
Developed in honor of scalability, simplicity and efficiency.

Essentially, it is peer-to-service framework, which allows you to create multiple services (roles) on single peer and communicate to them with ease.

Roletalk internally uses Websocket for data transferring as TCP framing tool with minimal network overhead.
Currently there is corresponding wire-compatible JS framework.

## <a name='Usecase'></a> Use case

Choose roletalk if you need to build brokerless architecture. Such approach brings some advantages: minimal RTT (round-trip time), less network transferring, no SPOF (single point of failure). In some scenarios broker can be a bottleneck also.

Roletalk lets you implement flexible communication patterns:

- Publish-Subscribe,
- Remote Procedure Call (RPC),
- Event (single message with optional payload),
- Pipeline (sequental distributed data processing),
- Bus (many-to-many communication),
- Survey (request-reply to multiple peers).

## <a name='Features'></a> Features

• Client as service.
No matter who establish connection. Both listener and dialer of a connection are services.
It allows you to deploy services not exposing them to the Internet or behind private network etc.
Of course such instances should connect to listeners they have network access to.

• Automatic service-recognition. Just connect peers together and each of them will know each about other's roles (services), IDs, names and meta info.
If peer's role gets disabled or enabled each connected peer immediately gets informed and rebuilds its internal state.

• Redunant connections support. There could be multiple connections between two peers. All communication is load-balanced between them.
With auto-reconnection this can be useful to keep connected two peers which are in case of one peer's IP address is changed.

• Binary streams. Transfer large or unknown amounts of data via streams.

• Round-robin client-side load balancing between units implementing a role (service);

• No internal heavy message conversions (json/xml serialization/parsing). Just binary to utf-8/int and vice-versa.

• Optional TLS on transport layer.

• Scalable and simple in-built authentication: each peer can have zero or multiple ID:KEY combinations.

## <a name='Concept'></a> Concept

### <a name='Structure'></a> Structure

Roletalk concept consists of:

- <b>Peer</b> - local node in your peer-to-peer architecture. Peer can have Units, Roles and Destinations.
- <b>Unit</b> - remote node connected to the Peer
- <b>Connection</b> - communication link between Peer and Unit. Each Unit can have one or more connections. If there are redunant connection, communications is load-balanced between them: checking which of them is not busy or choosing random one if no vacant one found. In case of last connection gets aborted Unit gets closed and removed from the Peer
- <b>Role</b> - service registered on Peer. Role can have handlers for each communication type. Role can be <b>active</b> or <b>inactive</b>. When role changes its state all connected units get informed to rebuild their state
- <b>Destination</b> - role registered on Units. Destination includes all connected units which serve corresponding role. If last Unit gets disconnected or disables the role, Destinations gets closed.
All communications is performed via Destinations's methods and is load-balanced among its Units unless Unit is explicitly specified. Unit is chosen for each message / request / stream. 

### <a name='Communication'></a> Communication

Roletalk defines three types of communication:

- <b>Message</b> - one-way act of communication. Should be used when no delivery acknowledgement is needed. Successfully sent message means that it has been written to underlying socket
- <b>Request</b> - request in common meaning. Request can only be rejected or replied. Returns error when Unit rejects it, timeout exceeds or Unit disconnects after request was sent.
- <b>Stream</b> - one-way stream of binary data. Streams can be <b>Readable</b> and <b>Writable</b>. If Peer calls Readable (`Destination.Readable()`) then Units handle Writable (`Role.OnWritable()`) and vice-versa. Stream sessions begin with Request. After Unit replied for request, data is transferred over connection used for the reply. If  connection aborts stream destroys.

Incoming messages are wrapped in <b>Context</b> - object with payload and meta info for all types of incoming messages (message, request, request for stream). 

All communication is performed with two basic properties: <b>Role</b> and <b>Event</b> (name of action. Some synonyms in other frameworks: method, path, action) to identify which handler to call.
When Unit gets message it forwards it to corresponding role or rejects it if such role isn't specified on the Peer. If role has no handlers for event it rejects it, otherwise call handlers.

## <a name='Datatypes'></a> Data types

Roletalk uses six data types:

- <b>Binary</b> - `[]byte`
- <b>Null</b> - `nil`
- <b>Bool</b> - `bool`
- <b>String</b> - `string`
- <b>Number</b> - `float64`
- <b>Object</b> - `[]byte` of JSON-stringified object

All communication (except stream sessions) can use data of any type. Type of data should be chosen on stage of designing microservice specification or retrieved by calling Context methods.

### <a name='Acquaintance'></a> Acquaintance

One aditional feature of roletalk is acquaintance. It is could be considered as redunant but it nicely fits the use case of frameworks.

The process of it is simple. You have listener PEER_A which is connected to listener PEER_B. After some PEER_C has connected to PEER_A, PEER_A will send to PEER_C info {ID, ADDRESS, ROLES} of PEER_A. If PEER_C is FRIENDLY, has one or more destination from received ROLES and PEER_C is not connected to peer with provided ID it will start connection to the ADDRESS

Acquantance is enabled by default, but you can set Friendly option to FALSE to disable is.

## <a name='Security'></a> Security

To achieve strong MITM-protection use HTTPS.

Framework is provided with simple in-built authentication mechanism: preshared ID:KEY pairs. Peers which are supposed to be connected should have at least one common ID:KEY pair.

The authentication process between two peers can be described in a simple few steps:

1. Each peer sends a list of his auth ID's and randomly generated CHALLENGE to another. If no auth ID:KEY specified peer sends auth confirmation.

2. Each peer checks if locally registered list of ID's intersects with the one received from remote peer. If no ID matched, sends error and closes conenctions with corresponding code.

3. Each peer chooses random intersected ID and hashes received CHALLENGE with corresponding KEY for that ID. Sends hashed RESPONSE value in pair with that ID.

4. Each peer receives RESPONSE with ID and checks if CHALLENGE hashed with corresponding KEY of received ID equals received RESPONSE. In such case sends auth confirmation. Else rejects auth and closes connection.

5. In case of both peers confirm each others's RESPONSES the handshake process is complete and then connections take part in futher communication. If auth exceeds timeout or some peer sends error and/or closes connection handshake fails.

## <a name='Contribution'></a> Contribution

Project is MIT-licensed. 

Feel free to open issues and fork. 

If you have any ideas or remarks you are welcome to contact the author.

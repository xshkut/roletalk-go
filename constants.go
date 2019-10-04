package roletalk

import "time"

type messageType byte

const (
	//restrictions
	maxCorrelation  correlation = 1<<53 - 1
	protocolVersion string      = "1.0.0"
	//timing
	authTimeot        time.Duration = 5 * time.Second
	heartBeatTimeout  time.Duration = 5 * time.Second
	heartBeatInterval time.Duration = 10 * time.Second
	reconnInterval    time.Duration = 15 * time.Second
	requestTimeout    time.Duration = 10 * time.Minute

	//protocol

	//first byte
	byteError         byte = 0
	byteAuthChallenge byte = 1
	byteAuthResponse  byte = 2
	byteAuthConfirmed byte = 3

	typeMessage       byte = 100
	typeRequest       byte = 101
	typeResolve       byte = 102
	typeReader        byte = 103
	typeWriter        byte = 104
	typeReject        byte = 105
	typeStreamData    byte = 106
	typeStreamResolve byte = 107
	typeStreamReject  byte = 108

	typeAcquaint byte = 200
	typeRoles    byte = 201

	//stream control byte
	streamByteChunk  byte = 0
	streamByteFinish byte = 1
	streamByteError  byte = 2
	streamByteQuota  byte = 3
	//protocol close codes
	errManualClose                 = 4000
	errAuthRejected                = 4001
	errWrongMessageType            = 4002
	errWrongDataType               = 4003
	errWrongCorrelation            = 4004
	errHeartbeatTimeout            = 4005
	errIncorrectMessageStructure   = 4006
	errIncompatibleProtocolVersion = 4007

	//errorMessages
	errStrOnceResponded     = "Already responded"
	errNonPositiveWaterMark = "Cannot set HighWaterMark <= 0"
	errConnClosed           = "Connection closed"

	// defaults
	defQuotaThreshold float64 = 0.66
	defQuotaSizeBytes         = 1024 * 16
)

package roletalk

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"

	"gotest.tools/assert"
)

func TestBinaryConversion(t *testing.T) {
	t.Run("Generate challenge with list of id's", testGenerateChallengeWithIds)
	t.Run("Serialize int to bytes", testSerializeInt)
	t.Run("Serialize correlation to bytes", testSerializeCorrelation)
	t.Run("Serialize INT to 2 bytes", testInt2Bytes)
	t.Run("Parse slice to int", testsliceToCorrelation)
	t.Run("Serialize one-way message", testSerializeOneWay)
	t.Run("parse one-way message", testSerializeOneWay)
	t.Run("Serialize request message", testSerializeRequest)
	t.Run("Serialize stream request message", testSerializeStreamRequest)
	t.Run("parse request message", testParseRequest)
	t.Run("parse stream request message", testParseStreamRequest)
	t.Run("Serialize response message", testSerializeResponse)
	t.Run("Serialize stream response message", testSerializeStreamResponse)
	t.Run("parse response message", testParseResponse)
	t.Run("parse stream response message", testParseStreamResponse)
	t.Run("conversions to binary from different types", testMarkDataType)
	t.Run("testing semver compatibility", testSemverCompatibility)
}

func testSerializeCorrelation(t *testing.T) {
	assert.Assert(t, reflect.DeepEqual(serializeCorrelation(0), []byte{0}))
	assert.Assert(t, reflect.DeepEqual(serializeCorrelation(255), []byte{255}))
	assert.Assert(t, reflect.DeepEqual(serializeCorrelation(256), []byte{1, 0}))
	assert.Assert(t, reflect.DeepEqual(serializeCorrelation(correlation(math.Pow(256, 1))*100), []byte{100, 0}))
	assert.Assert(t, reflect.DeepEqual(serializeCorrelation(correlation(math.Pow(256, 1))*100+200), []byte{100, 200}))
	assert.Assert(t, reflect.DeepEqual(serializeCorrelation(correlation(math.Pow(256, 2))*10+correlation(math.Pow(256, 1))*50+255), []byte{10, 50, 255}))
}

func testSerializeInt(t *testing.T) {
	assert.Assert(t, reflect.DeepEqual(serializeInt(0), []byte{0}))
	assert.Assert(t, reflect.DeepEqual(serializeInt(255), []byte{255}))
	assert.Assert(t, reflect.DeepEqual(serializeInt(256), []byte{1, 0}))
	assert.Assert(t, reflect.DeepEqual(serializeInt(int(math.Pow(256, 1))*100), []byte{100, 0}))
	assert.Assert(t, reflect.DeepEqual(serializeInt(int(math.Pow(256, 1))*100+200), []byte{100, 200}))
	assert.Assert(t, reflect.DeepEqual(serializeInt(int(math.Pow(256, 2))*10+int(math.Pow(256, 1))*50+255), []byte{10, 50, 255}))
}

func testInt2Bytes(t *testing.T) {
	assert.Assert(t, reflect.DeepEqual(int2Bytes(256*30+40), []byte{30, 40}))
	assert.Assert(t, reflect.DeepEqual(int2Bytes(0), []byte{0, 0}))
	assert.Assert(t, reflect.DeepEqual(int2Bytes(1), []byte{0, 1}))
}

func testsliceToCorrelation(t *testing.T) {
	assert.Equal(t, sliceToCorrelation([]byte{30, 40, 111}), correlation(30*256*256+40*256+111))
	assert.Equal(t, sliceToCorrelation([]byte{30, 40}), correlation(30*256+40))
	assert.Equal(t, sliceToCorrelation([]byte{0, 0}), correlation(0))
	assert.Equal(t, sliceToCorrelation([]byte{0, 1}), correlation(1))
}

func testSerializeOneWay(t *testing.T) {
	data, err := markDataType("data")
	if err != nil {
		t.Error(err)
	}
	compared := []byte{typeMessage, 0, 4, 0, 5, 114, 111, 108, 101, 101, 118, 101, 110, 116, 3, 100, 97, 116, 97}
	assert.Assert(t, reflect.DeepEqual(serializeOneway("role", "event", data), compared))
}

func testSerializeRequest(t *testing.T) {
	data, err := markDataType("data")
	if err != nil {
		t.Error(err)
	}
	compared := []byte{typeRequest, 0, 4, 0, 5, 1, 114, 111, 108, 101, 101, 118, 101, 110, 116, 50, 3, 100, 97, 116, 97}
	assert.Assert(t, reflect.DeepEqual(serializeRequest("role", "event", 50, data), compared))
}

func testSerializeStreamRequest(t *testing.T) {
	data, err := markDataType("data")
	if err != nil {
		t.Error(err)
	}
	compared := []byte{typeReader, 0, 4, 0, 5, 1, 1, 114, 111, 108, 101, 101, 118, 101, 110, 116, 50, 99, 3, 100, 97, 116, 97}
	assert.Assert(t, reflect.DeepEqual(serializeStreamRequest(typeReader, "role", "event", 50, 99, data), compared))
}

func testSerializeResponse(t *testing.T) {
	data, err := markDataType("data")
	if err != nil {
		t.Error(err)
	}
	compared := []byte{typeResolve, 1, 50, 3, 100, 97, 116, 97}
	assert.Assert(t, reflect.DeepEqual(serializeResponse(typeResolve, 50, data), compared))
}

func testSerializeStreamResponse(t *testing.T) {
	data, err := markDataType("data")
	if err != nil {
		t.Error(err)
	}
	compared := []byte{typeStreamResolve, 1, 1, 50, 150, 3, 100, 97, 116, 97}
	assert.Assert(t, reflect.DeepEqual(serializeStreamResponse(typeStreamResolve, 50, 150, data), compared))
}

func testParseOneWay(t *testing.T) {
	data, err := markDataType("data")
	if err != nil {
		t.Error(err)
	}
	role, event, tp, payload := parseOneway([]byte{0, 4, 0, 5, 114, 111, 108, 101, 101, 118, 101, 110, 116, 3, 100, 97, 116, 97})
	assert.Assert(t, reflect.DeepEqual(payload, data[1:]))
	assert.Equal(t, role, "role")
	assert.Equal(t, event, "event")
	assert.Equal(t, tp, DatatypeString)
}

func testParseRequest(t *testing.T) {
	data, err := markDataType("data")
	if err != nil {
		t.Error(err)
	}
	role, event, cor, tp, payload := parseRequest([]byte{0, 4, 0, 5, 1, 114, 111, 108, 101, 101, 118, 101, 110, 116, 77, 3, 100, 97, 116, 97})
	assert.Assert(t, reflect.DeepEqual(payload, data[1:]))
	assert.Equal(t, role, "role")
	assert.Equal(t, event, "event")
	assert.Equal(t, cor, correlation(77))
	assert.Equal(t, tp, DatatypeString)
}

func testParseStreamRequest(t *testing.T) {
	data, err := markDataType("data")
	if err != nil {
		t.Error(err)
	}
	role, event, cor, channel, tp, payload := parseStreamRequest([]byte{0, 4, 0, 5, 1, 1, 114, 111, 108, 101, 101, 118, 101, 110, 116, 77, 88, 3, 100, 97, 116, 97})
	assert.Assert(t, reflect.DeepEqual(payload, data[1:]))
	assert.Equal(t, role, "role")
	assert.Equal(t, event, "event")
	assert.Equal(t, cor, correlation(77))
	assert.Equal(t, channel, correlation(88))
	assert.Equal(t, tp, DatatypeString)
}

func testParseResponse(t *testing.T) {
	data, err := markDataType("data")
	if err != nil {
		t.Error(err)
	}
	cor, tp, payload := parseResponse([]byte{1, 77, 3, 100, 97, 116, 97})
	assert.Assert(t, reflect.DeepEqual(payload, data[1:]))
	assert.Equal(t, cor, correlation(77))
	assert.Equal(t, tp, DatatypeString)
}

func testParseStreamResponse(t *testing.T) {
	data, err := markDataType("data")
	if err != nil {
		t.Error(err)
	}
	cor, channel, tp, payload := parseStreamResponse([]byte{1, 1, 77, 88, 3, 100, 97, 116, 97})
	assert.Assert(t, reflect.DeepEqual(payload, data[1:]))
	assert.Equal(t, cor, correlation(77))
	assert.Equal(t, channel, correlation(88))
	assert.Equal(t, tp, DatatypeString)
}

func testGenerateChallengeWithIds(t *testing.T) {
	peer := NewPeer(PeerOptions{})
	peer.AddKey("some_id", "some_key")
	result, challenge, _ := peer.generateChallengeWithIds()
	expected := `{"challenge":"` + challenge + `","ids":["some_id"]}`
	assert.Equal(t, result, expected)
}

type testConv struct {
	descr     string
	compared  interface{}
	hardcoded []byte
}

type testStruct struct {
	A int
	B string
	C bool
}

func testMarkDataType(t *testing.T) {
	tests := []testConv{}
	// tests = append(tests, testConv{"binary", []byte{1, 2, 3, 4, 5}, []byte{0, 1, 2, 3, 4, 5}})
	tests = append(tests, testConv{"nil", nil, []byte{1}})
	tests = append(tests, testConv{"false", false, []byte{2, 0}})
	tests = append(tests, testConv{"true", true, []byte{2, 1}})
	tests = append(tests, testConv{"string", "some super string", append([]byte{3}, []byte("some super string")...)})
	tests = append(tests, testConv{"float", 123.1122334456, append([]byte{4}, []byte("123.1122334456")...)})
	tests = append(tests, testConv{"float", 123.1122, append([]byte{4}, []byte("123.1122")...)})
	d := testStruct{A: 1, B: "awd", C: true}
	hardcoded := append([]byte{5}, []byte(`{"A":1,"B":"awd","C":true}`)...)
	if converted, err := markDataType(d); err != nil {
		t.Fatal(err)
	} else {
		assert.Assert(t, reflect.DeepEqual(converted, hardcoded))
		if retrieved, e := retrieveDataByType(Datatype(hardcoded[0]), hardcoded[1:]); e != nil {
			t.Fatal(e)
		} else {
			if val, ok := retrieved.([]byte); ok == true {
				ts := testStruct{}
				if err := json.Unmarshal(val, &ts); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, ts, d)
			} else {
				t.Fatal("Wrong type")
			}
		}
	}
	for i, c := range tests {
		testForwardAndReverseConversion(t, c.compared, c.hardcoded, fmt.Sprintf("%d.%v", i+1, c.descr))
	}
}

func testForwardAndReverseConversion(t *testing.T, compared interface{}, hardcoded []byte, descr string) {
	t.Run(descr, func(t *testing.T) {
		if converted, err := markDataType(compared); err != nil {
			t.Fatal(err)
		} else {
			assert.Assert(t, reflect.DeepEqual(converted, hardcoded))
			if retrieved, e := retrieveDataByType(Datatype(hardcoded[0]), hardcoded[1:]); e != nil {
				t.Fatal(e)
			} else {
				assert.Equal(t, compared, retrieved)
			}
		}
	})
}

func testSemverCompatibility(t *testing.T) {
	assert.NilError(t, checkProtocolCompatibility("0.1.7", "0.9.4"))
	assert.NilError(t, checkProtocolCompatibility("1.1.7", "1.1.7"))
	assert.Error(t, checkProtocolCompatibility("1.1.7", "0.9.4"), errLocalVersionHigher)
	assert.Error(t, checkProtocolCompatibility("1.1.7", "2.9.4"), errRemoteVersionHigher)
	assert.ErrorContains(t, checkProtocolCompatibility("", "2.9.4"), errCantParseLocal)
	assert.ErrorContains(t, checkProtocolCompatibility("13", "2.9.4"), errCantParseLocal)
}

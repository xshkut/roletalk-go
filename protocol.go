package roletalk

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func parseString(inc []byte) string {
	return string(inc[1:])
}

func serializeString(t byte, str string) []byte {
	return append([]byte{t}, []byte(str)...)
}

func serializeOneway(role, event string, msg []byte) []byte {
	binRole := []byte(role)
	binEvent := []byte(event)
	temp := []byte{typeMessage}
	temp = append(temp, int2Bytes(len(binRole))...)
	temp = append(temp, int2Bytes(len(binEvent))...)
	temp = append(temp, binRole...)
	temp = append(temp, binEvent...)
	temp = append(temp, msg...)
	return temp
}

func serializeRequest(role, event string, corr correlation, msg []byte) []byte {
	binRole := []byte(role)
	binEvent := []byte(event)
	binCor := serializeCorrelation(corr)
	temp := []byte{typeRequest}
	temp = append(temp, int2Bytes(len(binRole))...)
	temp = append(temp, int2Bytes(len(binEvent))...)
	temp = append(temp, byte(len(binCor)))
	temp = append(temp, binRole...)
	temp = append(temp, binEvent...)
	temp = append(temp, binCor...)
	temp = append(temp, msg...)
	return temp
}

func serializeStreamRequest(t byte, role, event string, corr correlation, channel correlation, msg []byte) []byte {
	binRole := []byte(role)
	binEvent := []byte(event)
	binCor := serializeCorrelation(corr)
	binChan := serializeCorrelation(channel)
	temp := []byte{t}
	temp = append(temp, int2Bytes(len(binRole))...)
	temp = append(temp, int2Bytes(len(binEvent))...)
	temp = append(temp, byte(len(binCor)))
	temp = append(temp, byte(len(binChan)))
	temp = append(temp, binRole...)
	temp = append(temp, binEvent...)
	temp = append(temp, binCor...)
	temp = append(temp, binChan...)
	temp = append(temp, msg...)
	return temp
}

func serializeResponse(t byte, corr correlation, msg []byte) []byte {
	binCor := serializeCorrelation(corr)
	temp := []byte{t}
	temp = append(temp, byte(len(binCor)))
	temp = append(temp, binCor...)
	temp = append(temp, msg...)
	return temp
}

func serializeStreamResponse(t byte, corr, channel correlation, msg []byte) []byte {
	binCor := serializeCorrelation(corr)
	binChan := serializeCorrelation(channel)
	temp := []byte{t}
	temp = append(temp, byte(len(binCor)))
	temp = append(temp, byte(len(binChan)))
	temp = append(temp, binCor...)
	temp = append(temp, binChan...)
	temp = append(temp, msg...)
	return temp
}

func parseOneway(raw []byte) (role, event string, dataType Datatype, rowData []byte) {
	roleLen := sliceToCorrelation(raw[0:2])
	eventLen := sliceToCorrelation(raw[2:4])
	role = string(raw[4 : 4+roleLen])
	event = string(raw[4+roleLen : 4+roleLen+eventLen])
	dataType = Datatype(raw[4+roleLen+eventLen])
	rowData = raw[4+roleLen+eventLen+1:]
	return
}

func parseRequest(raw []byte) (role, event string, corr correlation, dataType Datatype, rowData []byte) {
	roleLen := sliceToCorrelation(raw[0:2])
	eventLen := sliceToCorrelation(raw[2:4])
	corLen := correlation(raw[4])
	role = string(raw[5 : 5+roleLen])
	event = string(raw[5+roleLen : 5+roleLen+eventLen])
	corr = sliceToCorrelation(raw[5+roleLen+eventLen : 5+roleLen+eventLen+corLen])
	dataType = Datatype(raw[5+roleLen+eventLen+corLen])
	rowData = raw[5+roleLen+eventLen+corLen+1:]
	return
}

func parseStreamRequest(raw []byte) (role, event string, corr, channel correlation, dataType Datatype, rowData []byte) {
	roleLen := sliceToCorrelation(raw[0:2])
	eventLen := sliceToCorrelation(raw[2:4])
	corLen := correlation(raw[4])
	chanLen := correlation(raw[5])
	roleFrom := 6
	eventFrom := roleFrom + int(roleLen)
	corrFrom := eventFrom + int(eventLen)
	chanFrom := corrFrom + int(corLen)
	typePos := chanFrom + int(chanLen)
	role = string(raw[roleFrom:eventFrom])
	event = string(raw[eventFrom:corrFrom])
	corr = sliceToCorrelation(raw[corrFrom:chanFrom])
	channel = sliceToCorrelation(raw[chanFrom:typePos])
	dataType = Datatype(raw[typePos])
	rowData = raw[typePos+1:]
	return
}

func parseResponse(raw []byte) (corr correlation, dataType Datatype, rowData []byte) {
	corLen := raw[0]
	corr = sliceToCorrelation(raw[1 : 1+corLen])
	dataType = Datatype(raw[1+corLen])
	rowData = raw[2+corLen:]
	return
}

func parseStreamResponse(raw []byte) (corr correlation, channel correlation, dataType Datatype, rowData []byte) {
	corLen := raw[0]
	chanLen := raw[1]
	corr = sliceToCorrelation(raw[2 : 2+corLen])
	channel = sliceToCorrelation(raw[2+corLen : 2+corLen+chanLen])
	dataType = Datatype(raw[2+chanLen+chanLen])
	rowData = raw[3+chanLen+chanLen:]
	return
}

func parseRoles(raw []byte) (roles rolesMsg, err error) {
	err = json.Unmarshal(raw, &roles)
	return
}

func markDataType(data interface{}) (result []byte, err error) {
	switch d := data.(type) {
	case []byte:
		result = append([]byte{0}, d...)
		return
	case nil:
		result = []byte{1}
		return
	case bool:
		if d == false {
			result = []byte{2, 0}
		} else {
			result = []byte{2, 1}
		}
		return
	case string:
		result = append([]byte{3}, []byte(d)...)
		return
	case float32:
		result = append([]byte{4}, strings.TrimRight(fmt.Sprintf("%.10f", d), "0")...)
		return
	case float64:
		result = append([]byte{4}, strings.TrimRight(fmt.Sprintf("%.10f", d), "0")...)
		return
	case int8:
		result = append([]byte{4}, fmt.Sprintf("%d", d)...)
		return
	case int16:
		result = append([]byte{4}, fmt.Sprintf("%d", d)...)
		return
	case int32:
		result = append([]byte{4}, fmt.Sprintf("%d", d)...)
		return
	case int64:
		result = append([]byte{4}, fmt.Sprintf("%d", d)...)
		return
	case int:
		result = append([]byte{4}, fmt.Sprintf("%d", d)...)
		return
	case uint8:
		result = append([]byte{4}, fmt.Sprintf("%d", d)...)
		return
	case uint16:
		result = append([]byte{4}, fmt.Sprintf("%d", d)...)
		return
	case uint32:
		result = append([]byte{4}, fmt.Sprintf("%d", d)...)
		return
	case uint64:
		result = append([]byte{4}, fmt.Sprintf("%d", d)...)
		return
	case uint:
		result = append([]byte{4}, fmt.Sprintf("%d", d)...)
		return
	case uintptr:
		result = append([]byte{4}, fmt.Sprintf("%d", d)...)
		return
	case complex64:
		result = append([]byte{3}, fmt.Sprintf("%.10f", d)...)
		return
	case complex128:
		result = append([]byte{3}, fmt.Sprintf("%.10f", d)...)
		return
	default:
		if jsoned, e := json.Marshal(d); e != nil {
			err = e
		} else {
			result = append([]byte{5}, jsoned...)
		}
		return
	}
}

func retrieveDataByType(t Datatype, raw []byte) (data interface{}, err error) {
	switch t {
	case DatatypeBinary:
		data = raw
	case DatatypeNull:
		data = nil
	case DatatypeBool:
		if raw[0] == 0 {
			data = false
			break
		}
		data = true
	case DatatypeString:
		data = string(raw)
	case DatatypeNumber:
		data, err = strconv.ParseFloat(string(raw), 64)
	case DatatypeJSON:
		data = raw
	default:
		return nil, fmt.Errorf("Unknown datatype: %v", t)
	}
	return
}

func int2Bytes(len int) []byte {
	result := make([]byte, 2)
	result[1] = byte(len % 256)
	result[0] = byte(len / 256)
	return result
}

func serializeCorrelation(value correlation) []byte {
	result := []byte{byte(value % 256)}
	for i := 1; correlation(math.Pow(256, float64(i))) <= correlation(value); i++ {
		result = append(result, byte(float64(value)/math.Pow(256, float64(i))))
	}
	if len := len(result); len > 1 {
		result1 := make([]byte, len)
		for i, val := range result {
			result1[len-1-i] = val
		}
		result = result1
	}
	return result
}

func serializeInt(value int) []byte {
	result := []byte{byte(value % 256)}
	for i := 1; int(math.Pow(256, float64(i))) <= value; i++ {
		result = append(result, byte(float64(value)/math.Pow(256, float64(i))))
	}
	if len := len(result); len > 1 {
		result1 := make([]byte, len)
		for i, val := range result {
			result1[len-1-i] = val
		}
		result = result1
	}
	return result
}

func sliceToCorrelation(sl []byte) correlation {
	var res correlation
	len := len(sl)
	for i, b := range sl {
		res += correlation(b) * correlation(math.Pow(256, float64(len-1-i)))
	}
	return res
}

func sliceToInt(sl []byte) int {
	var res int
	len := len(sl)
	for i, b := range sl {
		res += int(b) * int(math.Pow(256, float64(len-1-i)))
	}
	return res
}

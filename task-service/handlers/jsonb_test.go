package handlers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONB_Scan_Nil(t *testing.T) {
	var j JSONB
	err := j.Scan(nil)
	assert.NoError(t, err)
	assert.Nil(t, j)
}

func TestJSONB_Scan_Bytes(t *testing.T) {
	var j JSONB
	err := j.Scan([]byte(`{"key":"value"}`))
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`{"key":"value"}`), j)
}

func TestJSONB_Scan_String(t *testing.T) {
	var j JSONB
	err := j.Scan(`{"name":"test"}`)
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`{"name":"test"}`), j)
}

func TestJSONB_Scan_InvalidType(t *testing.T) {
	var j JSONB
	err := j.Scan(12345)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected []byte or string")
}

func TestJSONB_Scan_EmptyBytes(t *testing.T) {
	var j JSONB
	err := j.Scan([]byte(""))
	assert.NoError(t, err)
	assert.Nil(t, j)
}

func TestJSONB_Scan_LeadingVersionByte(t *testing.T) {
	// PostgreSQL JSONB binary format may have leading \x01 byte
	data := append([]byte{0x01}, []byte(`{"key":"value"}`)...)
	var j JSONB
	err := j.Scan(data)
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`{"key":"value"}`), j)
}

func TestJSONB_Scan_LeadingNullByte(t *testing.T) {
	data := append([]byte{0x00}, []byte(`{"key":"value"}`)...)
	var j JSONB
	err := j.Scan(data)
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`{"key":"value"}`), j)
}

func TestJSONB_Scan_InvalidJSON(t *testing.T) {
	var j JSONB
	err := j.Scan([]byte(`{not valid json}`))
	assert.NoError(t, err)
	assert.Nil(t, j) // Invalid JSON results in nil, no error
}

func TestJSONB_Scan_Array(t *testing.T) {
	var j JSONB
	err := j.Scan([]byte(`[1,2,3]`))
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`[1,2,3]`), j)
}

func TestJSONB_Scan_StringValue(t *testing.T) {
	var j JSONB
	err := j.Scan([]byte(`"hello"`))
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`"hello"`), j)
}

func TestJSONB_Scan_Number(t *testing.T) {
	var j JSONB
	err := j.Scan([]byte(`42`))
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`42`), j)
}

func TestJSONB_Scan_NegativeNumber(t *testing.T) {
	var j JSONB
	err := j.Scan([]byte(`-3.14`))
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`-3.14`), j)
}

func TestJSONB_Scan_BoolTrue(t *testing.T) {
	var j JSONB
	err := j.Scan([]byte(`true`))
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`true`), j)
}

func TestJSONB_Scan_BoolFalse(t *testing.T) {
	var j JSONB
	err := j.Scan([]byte(`false`))
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`false`), j)
}

func TestJSONB_Scan_NullLiteral(t *testing.T) {
	var j JSONB
	err := j.Scan([]byte(`null`))
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`null`), j)
}

// ==================== Value Tests ====================

func TestJSONB_Value_Nil(t *testing.T) {
	var j JSONB
	val, err := j.Value()
	assert.NoError(t, err)
	assert.Nil(t, val)
}

func TestJSONB_Value_Valid(t *testing.T) {
	j := JSONB(`{"key":"value"}`)
	val, err := j.Value()
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"key":"value"}`), val)
}

// ==================== MarshalJSON Tests ====================

func TestJSONB_MarshalJSON_Nil(t *testing.T) {
	var j JSONB
	data, err := j.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, []byte("null"), data)
}

func TestJSONB_MarshalJSON_Empty(t *testing.T) {
	j := JSONB{}
	data, err := j.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, []byte("null"), data)
}

func TestJSONB_MarshalJSON_Valid(t *testing.T) {
	j := JSONB(`{"key":"value"}`)
	data, err := j.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"key":"value"}`), data)
}

func TestJSONB_MarshalJSON_InvalidJSON(t *testing.T) {
	j := JSONB(`{broken`)
	data, err := j.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, []byte("null"), data) // Returns null for invalid JSON
}

func TestJSONB_MarshalJSON_InStruct(t *testing.T) {
	type testStruct struct {
		Data JSONB `json:"data"`
	}

	s := testStruct{Data: JSONB(`{"nested":"value"}`)}
	data, err := json.Marshal(s)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"data":{"nested":"value"}}`, string(data))
}

func TestJSONB_MarshalJSON_NilInStruct(t *testing.T) {
	type testStruct struct {
		Data JSONB `json:"data"`
	}

	s := testStruct{Data: nil}
	data, err := json.Marshal(s)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"data":null}`, string(data))
}

// ==================== UnmarshalJSON Tests ====================

func TestJSONB_UnmarshalJSON_Null(t *testing.T) {
	var j JSONB
	err := j.UnmarshalJSON([]byte("null"))
	assert.NoError(t, err)
	assert.Nil(t, j)
}

func TestJSONB_UnmarshalJSON_NilData(t *testing.T) {
	var j JSONB
	err := j.UnmarshalJSON(nil)
	assert.NoError(t, err)
	assert.Nil(t, j)
}

func TestJSONB_UnmarshalJSON_Valid(t *testing.T) {
	var j JSONB
	err := j.UnmarshalJSON([]byte(`{"key":"value"}`))
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`{"key":"value"}`), j)
}

func TestJSONB_UnmarshalJSON_InStruct(t *testing.T) {
	type testStruct struct {
		Data JSONB `json:"data"`
	}

	var s testStruct
	err := json.Unmarshal([]byte(`{"data":{"nested":"value"}}`), &s)
	assert.NoError(t, err)
	assert.Equal(t, JSONB(`{"nested":"value"}`), s.Data)
}

// ==================== Roundtrip Tests ====================

func TestJSONB_Roundtrip_ScanThenValue(t *testing.T) {
	original := `{"key":"value","num":42}`

	var j JSONB
	err := j.Scan([]byte(original))
	assert.NoError(t, err)

	val, err := j.Value()
	assert.NoError(t, err)
	assert.Equal(t, []byte(original), val)
}

func TestJSONB_Roundtrip_MarshalUnmarshal(t *testing.T) {
	original := JSONB(`{"key":"value","arr":[1,2,3]}`)

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	var result JSONB
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, original, result)
}

package nibe

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Device struct {
	Index     uint      `json:"deviceIndex"`
	ModeAid   ModeAid   `json:"aidMode"`
	ModeSmart ModeSmart `json:"smartMode"`
	Product   Product   `json:"product"`
}

type Product struct {
	Serial       string `json:"serialNumber"`
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	FirmwareID   string `json:"firmwareId"`
}

type ModeAid string

const (
	ModeAidInvalid ModeAid = ""
	ModeAidOn      ModeAid = "on"
	ModeAidOff     ModeAid = "off"
)

type ModeSmart string

const (
	ModeSmartInvalid ModeSmart = ""
	ModeSmartNormal  ModeSmart = "normal"
	ModeSmartAway    ModeSmart = "away"
)

type Point struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Metadata    Metadata `json:"metadata"`
	Value       Value    `json:"value"`
}

func (p *Point) UnmarshalJSON(data []byte) error {
	type shadow Point
	var res shadow

	err := json.Unmarshal(data, &res)
	if err != nil {
		return err
	}

	res.Title = strings.ReplaceAll(res.Title, "\u00ad", "")
	*p = Point(res)

	return nil
}

type Value struct {
	Type ValueType `json:"type"`
	OK   bool      `json:"isOK"`

	VariableID int `json:"variableId"`

	Int int    `json:"integerValue"`
	Str string `json:"stringValue"`
}

func (v Value) patchRequest() map[string]any {
	return map[string]any{
		"type":         v.Type,
		"variableId":   v.VariableID,
		"integerValue": v.Int,
	}
}

type Metadata struct {
	Type MetadataType `json:"type"`

	VariableID   int          `json:"variableId"`
	VariableType VariableType `json:"variableType"`
	VariableSize VariableSize `json:"variableSize"`

	Unit      string `json:"unit"`
	ShortUnit string `json:"shortUnit"`

	RegisterID   int      `json:"modbusRegisterID"`
	RegisterType Register `json:"modbusRegisterType"`
	Writable     bool     `json:"isWritable"`

	Divisor int `json:"divisor"`
	Decimal int `json:"decimal"`
	Min     int `json:"minValue"`
	Max     int `json:"maxValue"`
	Change  int `json:"change"`

	DefaultValueInt int    `json:"intDefaultValue"`
	DefaultValueStr string `json:"stringDefaultValue"`
}

type VariableType string

const (
	VariableTypeInvalid VariableType = ""
	VariableTypeBinary  VariableType = "binary"
	VariableTypeDate    VariableType = "date"
	VariableTypeFloat   VariableType = "floating-point"
	VariableTypeInteger VariableType = "integer"
	VariableTypeString  VariableType = "string"
	VariableTypeTime    VariableType = "time"
	VariableTypeUnknown VariableType = "unknown"
)

type VariableSize string

const (
	VariableSizeInvalid VariableSize = ""
	VariableSizeFloat32 VariableSize = "f4"
	VariableSizeFloat64 VariableSize = "f8"
	VariableSizeInt8    VariableSize = "s8"
	VariableSizeInt16   VariableSize = "s16"
	VariableSizeInt32   VariableSize = "s32"
	VariableSizeUint8   VariableSize = "u8"
	VariableSizeUint16  VariableSize = "u16"
	VariableSizeUint32  VariableSize = "u32"
	VariableSizeUnknown VariableSize = "unknown"
)

type Register string

const (
	RegisterInvalid Register = ""
	RegisterInput   Register = "MODBUS_INPUT_REGISTER"
	RegisterHolding Register = "MODBUS_HOLDING_REGISTER"
	RegisterNo      Register = "MODBUS_NO_REGISTER"
	RegisterUnknown Register = "ERR_UNKNOWN"
)

type MetadataType string

const (
	MetadataTypeInvalid MetadataType = ""
	MetadataTypeMetdata MetadataType = "metadata"
)

type ValueType string

const (
	ValueTypeInvalid ValueType = ""
	ValueTypeData    ValueType = "datavalue"
)

type Notification struct {
	ID          int    `json:"alarmId"`
	Description string `json:"description"`
	Header      string `json:"header"`
	Severity    int    `json:"severity"`
	Time        string `json:"time"`
	EquipName   string `json:"equipName"`
}

type APIError struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
}

func (a *APIError) String() string {
	return fmt.Sprintf("code: %d, message: %s", a.Code, a.Message)
}

func (a *APIError) Error() string {
	return a.String()
}

// Code generated by "enumer -type DataType -trimprefix DataType"; DO NOT EDIT.

package schema

import (
	"fmt"
)

const _DataTypeName = "UndefinedBaseArrayEnumRangeCompositeDomain"

var _DataTypeIndex = [...]uint8{0, 9, 13, 18, 22, 27, 36, 42}

func (i DataType) String() string {
	if i < 0 || i >= DataType(len(_DataTypeIndex)-1) {
		return fmt.Sprintf("DataType(%d)", i)
	}
	return _DataTypeName[_DataTypeIndex[i]:_DataTypeIndex[i+1]]
}

var _DataTypeValues = []DataType{0, 1, 2, 3, 4, 5, 6}

var _DataTypeNameToValueMap = map[string]DataType{
	_DataTypeName[0:9]:   0,
	_DataTypeName[9:13]:  1,
	_DataTypeName[13:18]: 2,
	_DataTypeName[18:22]: 3,
	_DataTypeName[22:27]: 4,
	_DataTypeName[27:36]: 5,
	_DataTypeName[36:42]: 6,
}

// DataTypeString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func DataTypeString(s string) (DataType, error) {
	if val, ok := _DataTypeNameToValueMap[s]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to DataType values", s)
}

// DataTypeValues returns all values of the enum
func DataTypeValues() []DataType {
	return _DataTypeValues
}

// IsADataType returns "true" if the value is listed in the enum definition. "false" otherwise
func (i DataType) IsADataType() bool {
	for _, v := range _DataTypeValues {
		if i == v {
			return true
		}
	}
	return false
}

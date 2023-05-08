// Code generated by "enumer -type ConstraintType -trimprefix ConstraintType -json"; DO NOT EDIT.

package schema

import (
	"encoding/json"
	"fmt"
)

const _ConstraintTypeName = "UndefinedPKFKUniqueCheckTriggerExclusion"

var _ConstraintTypeIndex = [...]uint8{0, 9, 11, 13, 19, 24, 31, 40}

func (i ConstraintType) String() string {
	if i < 0 || i >= ConstraintType(len(_ConstraintTypeIndex)-1) {
		return fmt.Sprintf("ConstraintType(%d)", i)
	}
	return _ConstraintTypeName[_ConstraintTypeIndex[i]:_ConstraintTypeIndex[i+1]]
}

var _ConstraintTypeValues = []ConstraintType{0, 1, 2, 3, 4, 5, 6}

var _ConstraintTypeNameToValueMap = map[string]ConstraintType{
	_ConstraintTypeName[0:9]:   0,
	_ConstraintTypeName[9:11]:  1,
	_ConstraintTypeName[11:13]: 2,
	_ConstraintTypeName[13:19]: 3,
	_ConstraintTypeName[19:24]: 4,
	_ConstraintTypeName[24:31]: 5,
	_ConstraintTypeName[31:40]: 6,
}

// ConstraintTypeString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func ConstraintTypeString(s string) (ConstraintType, error) {
	if val, ok := _ConstraintTypeNameToValueMap[s]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to ConstraintType values", s)
}

// ConstraintTypeValues returns all values of the enum
func ConstraintTypeValues() []ConstraintType {
	return _ConstraintTypeValues
}

// IsAConstraintType returns "true" if the value is listed in the enum definition. "false" otherwise
func (i ConstraintType) IsAConstraintType() bool {
	for _, v := range _ConstraintTypeValues {
		if i == v {
			return true
		}
	}
	return false
}

// MarshalJSON implements the json.Marshaler interface for ConstraintType
func (i ConstraintType) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface for ConstraintType
func (i *ConstraintType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("ConstraintType should be a string, got %s", data)
	}

	var err error
	*i, err = ConstraintTypeString(s)
	return err
}

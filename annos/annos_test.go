package annos

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type AnnosTestSuite struct {
	suite.Suite
}

func (a *AnnosTestSuite) TestToMapInts() {
	// -- Given
	//
	given := Ints{
		Int:    1,
		Int8:   2,
		Int16:  3,
		Int32:  4,
		Int64:  5,
		UInt:   6,
		UInt8:  7,
		UInt16: 8,
		UInt32: 9,
		UInt64: 10,
	}

	given.IntPtr = &given.Int
	given.Int8Ptr = &given.Int8
	given.Int16Ptr = &given.Int16
	given.Int32Ptr = &given.Int32
	given.Int64Ptr = &given.Int64
	given.UIntPtr = &given.UInt
	given.UInt8Ptr = &given.UInt8
	given.UInt16Ptr = &given.UInt16
	given.UInt32Ptr = &given.UInt32
	given.UInt64Ptr = &given.UInt64

	expected := map[string]string{
		"canary.kage.cloud/int":          "1",
		"canary.kage.cloud/int_8":        "2",
		"canary.kage.cloud/int_16":       "3",
		"canary.kage.cloud/int_32":       "4",
		"canary.kage.cloud/int_64":       "5",
		"canary.kage.cloud/u_int":        "6",
		"canary.kage.cloud/u_int_8":      "7",
		"canary.kage.cloud/u_int_16":     "8",
		"canary.kage.cloud/u_int_32":     "9",
		"canary.kage.cloud/u_int_64":     "10",
		"canary.kage.cloud/int_ptr":      "1",
		"canary.kage.cloud/int_8_ptr":    "2",
		"canary.kage.cloud/int_16_ptr":   "3",
		"canary.kage.cloud/int_32_ptr":   "4",
		"canary.kage.cloud/int_64_ptr":   "5",
		"canary.kage.cloud/u_int_ptr":    "6",
		"canary.kage.cloud/u_int_8_ptr":  "7",
		"canary.kage.cloud/u_int_16_ptr": "8",
		"canary.kage.cloud/u_int_32_ptr": "9",
		"canary.kage.cloud/u_int_64_ptr": "10",
	}

	// -- When
	//
	actual := ToMap("canary.kage.cloud", given)

	// -- Then
	//
	a.Equal(expected, actual)
}

func (a *AnnosTestSuite) TestToMapFloats() {
	// -- Given
	//
	given := Floats{
		Float32: 1,
		Float64: 2,
	}

	given.Float32Ptr = &given.Float32
	given.Float64Ptr = &given.Float64

	expected := map[string]string{
		"canary.kage.cloud/float_32":     "1",
		"canary.kage.cloud/float_64":     "2",
		"canary.kage.cloud/float_32_ptr": "1",
		"canary.kage.cloud/float_64_ptr": "2",
	}

	// -- When
	//
	actual := ToMap("canary.kage.cloud", given)

	// -- Then
	//
	a.Equal(expected, actual)
}

func (a *AnnosTestSuite) TestToMapBools() {
	// -- Given
	//
	given := Bools{
		Bool: true,
	}

	given.BoolPtr = &given.Bool

	expected := map[string]string{
		"canary.kage.cloud/bool":     "true",
		"canary.kage.cloud/bool_ptr": "true",
	}

	// -- When
	//
	actual := ToMap("canary.kage.cloud", given)

	// -- Then
	//
	a.Equal(expected, actual)
}

func (a *AnnosTestSuite) TestToMapStrings() {
	// -- Given
	//
	given := Strings{
		String: "str",
	}

	given.StringPtr = &given.String

	expected := map[string]string{
		"canary.kage.cloud/string":     "str",
		"canary.kage.cloud/string_ptr": "str",
	}

	// -- When
	//
	actual := ToMap("canary.kage.cloud", given)

	// -- Then
	//
	a.Equal(expected, actual)
}

func (a *AnnosTestSuite) TestToMapMaps() {
	// -- Given
	//
	given := Maps{
		MapStringString: map[string]string{
			"str": "str",
		},
		MapStringInt: map[string]int{
			"str": 1,
		},
		MapStringFloat32: map[string]float32{
			"str": 1.1,
		},
		MapStringFloat64: map[string]float64{
			"str": 1.1,
		},
		MapStringBool: map[string]bool{
			"str": true,
		},
		MapStringMapStringString: map[string]map[string]string{
			"str": {
				"str": "str",
			},
		},
	}

	expected := map[string]string{
		"canary.kage.cloud/map_string_string":            "str=str",
		"canary.kage.cloud/map_string_int":               "str=1",
		"canary.kage.cloud/map_string_float_32":          "str=1.1",
		"canary.kage.cloud/map_string_float_64":          "str=1.1",
		"canary.kage.cloud/map_string_bool":              "str=true",
		"canary.kage.cloud/map_string_map_string_string": "str=str=str",
	}

	// -- When
	//
	actual := ToMap("canary.kage.cloud", given)

	// -- Then
	//
	a.Equal(expected, actual)
}

func (a *AnnosTestSuite) TestToMapSlices() {
	// -- Given
	//
	given := Slices{
		Strings: []string{"1", "2", "3"},
		Ints:    []int{1, 2, 3},
		Floats:  []float32{1, 2, 3},
		Bools:   []bool{true, false},
	}

	expected := map[string]string{
		"canary.kage.cloud/strings": "1,2,3",
		"canary.kage.cloud/ints":    "1,2,3",
		"canary.kage.cloud/floats":  "1,2,3",
		"canary.kage.cloud/bools":   "true,false",
	}

	// -- When
	//
	actual := ToMap("canary.kage.cloud", given)

	// -- Then
	//
	a.Equal(expected, actual)
}

func (a *AnnosTestSuite) TestToMapNested() {
	// -- Given
	//
	given := Nested{
		Bools: Bools{
			Bool: true,
		},
		Strings: &Strings{
			String: "str",
		},
		Map: map[string]string{
			"str": "str",
		},
		SubNested: SubNested{
			Slices: Slices{
				Floats: []float32{1},
			},
		},
	}

	expected := map[string]string{
		"canary.kage.cloud/bools/bool":                "true",
		"canary.kage.cloud/strings/string":            "str",
		"canary.kage.cloud/sub_nested/slices/floats":  "1",
		"canary.kage.cloud/sub_nested/slices/bools":   "",
		"canary.kage.cloud/sub_nested/slices/ints":    "",
		"canary.kage.cloud/sub_nested/slices/strings": "",
		"canary.kage.cloud/map":                       "str=str",
	}

	// -- When
	//
	actual := ToMap("canary.kage.cloud", given)

	// -- Then
	//
	a.Equal(expected, actual)
}

func (a *AnnosTestSuite) TestToMapNoDomain() {
	// -- Given
	//
	given := new(Bools)
	expected := map[string]string{
		"bool": "false",
	}

	// -- When
	//
	actual := ToMap("", given)

	// -- Then
	//
	a.Equal(expected, actual)
}

func (a *AnnosTestSuite) TestToMapNilGiven() {
	// -- Given
	//
	expected := map[string]string{}

	// -- When
	//
	actual := ToMap("", nil)

	// -- Then
	//
	a.Equal(expected, actual)
}

func (a *AnnosTestSuite) TestFromMapInts() {
	// -- Given
	//
	actual := new(Ints)
	given := map[string]string{
		"canary.kage.cloud/int":          "1",
		"canary.kage.cloud/int_8":        "2",
		"canary.kage.cloud/int_16":       "3",
		"canary.kage.cloud/int_32":       "4",
		"canary.kage.cloud/int_64":       "5",
		"canary.kage.cloud/u_int":        "6",
		"canary.kage.cloud/u_int_8":      "7",
		"canary.kage.cloud/u_int_16":     "8",
		"canary.kage.cloud/u_int_32":     "9",
		"canary.kage.cloud/u_int_64":     "10",
		"canary.kage.cloud/int_ptr":      "1",
		"canary.kage.cloud/int_8_ptr":    "2",
		"canary.kage.cloud/int_16_ptr":   "3",
		"canary.kage.cloud/int_32_ptr":   "4",
		"canary.kage.cloud/int_64_ptr":   "5",
		"canary.kage.cloud/u_int_ptr":    "6",
		"canary.kage.cloud/u_int_8_ptr":  "7",
		"canary.kage.cloud/u_int_16_ptr": "8",
		"canary.kage.cloud/u_int_32_ptr": "9",
		"canary.kage.cloud/u_int_64_ptr": "10",
		"canary.kage.c1oud/u_int_64_ptr": "10",
	}

	expected := &Ints{
		Int:    1,
		Int8:   2,
		Int16:  3,
		Int32:  4,
		Int64:  5,
		UInt:   6,
		UInt8:  7,
		UInt16: 8,
		UInt32: 9,
		UInt64: 10,
	}

	expected.IntPtr = &expected.Int
	expected.Int8Ptr = &expected.Int8
	expected.Int16Ptr = &expected.Int16
	expected.Int32Ptr = &expected.Int32
	expected.Int64Ptr = &expected.Int64
	expected.UIntPtr = &expected.UInt
	expected.UInt8Ptr = &expected.UInt8
	expected.UInt16Ptr = &expected.UInt16
	expected.UInt32Ptr = &expected.UInt32
	expected.UInt64Ptr = &expected.UInt64

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

func (a *AnnosTestSuite) TestFromMapFloats() {
	// -- Given
	//
	actual := new(Floats)
	expected := &Floats{
		Float32: 1,
		Float64: 2,
	}

	expected.Float32Ptr = &expected.Float32
	expected.Float64Ptr = &expected.Float64

	given := map[string]string{
		"canary.kage.cloud/float_32":     "1",
		"canary.kage.cloud/float_64":     "2",
		"canary.kage.cloud/float_32_ptr": "1",
		"canary.kage.cloud/float_64_ptr": "2",
		"canary.kage.c1oud/float_64_ptr": "2",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

func (a *AnnosTestSuite) TestFromMapBools() {
	// -- Given
	//
	actual := new(Bools)
	expected := &Bools{
		Bool: true,
	}

	f := false
	expected.BoolPtr = &f

	given := map[string]string{
		"canary.kage.cloud/bool":     "true",
		"canary.kage.cloud/bool_ptr": "false",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

func (a *AnnosTestSuite) TestFromMapStrings() {
	// -- Given
	//
	actual := new(Strings)
	expected := &Strings{
		String: "str",
	}

	f := "asd"
	expected.StringPtr = &f

	given := map[string]string{
		"canary.kage.cloud/string":     "str",
		"canary.kage.cloud/string_ptr": "asd",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

func (a *AnnosTestSuite) TestFromMapMaps() {
	// -- Given
	//
	actual := new(Maps)
	expected := &Maps{
		MapStringString: map[string]string{
			"str": "str",
		},
		MapStringInt: map[string]int{
			"str": 1,
		},
		MapStringFloat32: map[string]float32{
			"str": 1.1,
		},
		MapStringFloat64: map[string]float64{
			"str": 1.1,
		},
		MapStringBool: map[string]bool{
			"str": true,
		},
		MapStringMapStringString: map[string]map[string]string{
			"str": {
				"str": "str",
			},
		},
	}

	given := map[string]string{
		"canary.kage.cloud/map_string_string":            "str=str",
		"canary.kage.cloud/map_string_int":               "str=1",
		"canary.kage.cloud/map_string_float_32":          "str=1.1",
		"canary.kage.cloud/map_string_float_64":          "str=1.1",
		"canary.kage.cloud/map_string_bool":              "str=true",
		"canary.kage.cloud/map_string_map_string_string": "str=str=str",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

func (a *AnnosTestSuite) TestFromMapSlices() {
	// -- Given
	//
	actual := new(Slices)
	expected := &Slices{
		Strings: []string{"1", "2", "3"},
		Ints:    []int{1, 2, 3},
		Floats:  []float32{1, 2, 3},
		Bools:   []bool{true, false},
	}

	given := map[string]string{
		"canary.kage.cloud/strings": "1,2,3",
		"canary.kage.cloud/ints":    "1,2,3",
		"canary.kage.cloud/floats":  "1,2,3",
		"canary.kage.cloud/bools":   "true,false",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

func (a *AnnosTestSuite) TestFromMapNested() {
	// -- Given
	//
	actual := new(Nested)
	expected := &Nested{
		Bools: Bools{
			Bool: true,
		},
		Strings: &Strings{
			String: "str",
		},
		Map: map[string]string{
			"str": "str",
		},
		SubNested: SubNested{
			Slices: Slices{
				Floats: []float32{1},
			},
		},
	}

	given := map[string]string{
		"canary.kage.cloud/bools/bool":               "true",
		"canary.kage.cloud/strings/string":           "str",
		"canary.kage.cloud/sub_nested/slices/floats": "1",
		"canary.kage.cloud/map":                      "str=str",
		"canary.kage.cloud-map":                      "str=str",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

func (a *AnnosTestSuite) TestFromMapNestedStructKey() {
	// -- Given
	//
	actual := new(Nested)
	expected := &Nested{
		SubNested: SubNested{
			Slices: Slices{
				Floats: []float32{1},
			},
		},
	}

	given := map[string]string{
		"canary.kage.cloud/sub_nested": `{"slices": {"floats": [1]}}`,
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

func (a *AnnosTestSuite) TestFromMapHiddenNestedStructKey() {
	// -- Given
	//
	actual := new(Nested)
	expected := new(Nested)

	given := map[string]string{
		"canary.kage.cloud/sub_nested/hidden": "true",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

func (a *AnnosTestSuite) TestFromMapNil() {
	// -- When
	//
	err := FromMap("", map[string]string{}, nil)

	// -- Then
	//
	a.EqualError(err, "the annotation struct must not be nil")
}

func (a *AnnosTestSuite) TestFromMapNoDomain() {
	// -- Given
	//
	actual := new(Bools)
	expected := &Bools{
		Bool: true,
	}

	given := map[string]string{
		"bool":                       "true",
		"canary.kage.cloud/bool_ptr": "false",
	}

	// -- When
	//
	err := FromMap("", given, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

func (a *AnnosTestSuite) TestFromMapNoMatches() {
	// -- Given
	//
	actual := new(Bools)
	expected := &Bools{}

	given := map[string]string{
		"canary.kage.cloud/bool":     "true",
		"canary.kage.cloud/bool_ptr": "false",
	}

	// -- When
	//
	err := FromMap("anary.kage.cloud", given, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

func (a *AnnosTestSuite) TestFromMapBadBool() {
	// -- Given
	//
	actual := new(Bools)
	given := map[string]string{
		"canary.kage.cloud/bool": "1",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	a.EqualError(err, `failed to read canary.kage.cloud/bool: expected bool but got "1"`)
}

func (a *AnnosTestSuite) TestFromMapBadInt() {
	// -- Given
	//
	actual := new(Ints)
	given := map[string]string{
		"canary.kage.cloud/int": "1.1",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	a.EqualError(err, `failed to read canary.kage.cloud/int: expected int but got "1.1"`)
}

func (a *AnnosTestSuite) TestFromMapBadUInt() {
	// -- Given
	//
	actual := new(Ints)
	given := map[string]string{
		"canary.kage.cloud/u_int": "true",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	a.EqualError(err, `failed to read canary.kage.cloud/u_int: expected uint but got "true"`)
}

func (a *AnnosTestSuite) TestFromMapBadFloat() {
	// -- Given
	//
	actual := new(Floats)
	given := map[string]string{
		"canary.kage.cloud/float_32": "true",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	a.EqualError(err, `failed to read canary.kage.cloud/float_32: expected float32 but got "true"`)
}

func (a *AnnosTestSuite) TestFromMapBadMap() {
	// -- Given
	//
	actual := new(Maps)
	given := map[string]string{
		"canary.kage.cloud/map_string_bool": "str=1",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	a.EqualError(err, `failed to read canary.kage.cloud/map_string_bool: expected bool but got "1"`)
}

func (a *AnnosTestSuite) TestFromMapBadSlice() {
	// -- Given
	//
	actual := new(Slices)
	given := map[string]string{
		"canary.kage.cloud/bools": "1",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	a.EqualError(err, `failed to read canary.kage.cloud/bools: expected bool but got "1"`)
}

func (a *AnnosTestSuite) TestFromMapBadNestedStructKey() {
	// -- Given
	//
	actual := new(Nested)
	given := map[string]string{
		"canary.kage.cloud/sub_nested": `{"slices: {"floats": [1]}}`,
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, actual)

	// -- Then
	//
	a.EqualError(err, `failed to read canary.kage.cloud/sub_nested: invalid character 'f' after object key`)
}

func (a *AnnosTestSuite) TestFromMapNilStruct() {
	// -- Given
	//
	given := map[string]string{
		"canary.kage.cloud/bools": "1",
	}

	// -- When
	//
	err := FromMap("canary.kage.cloud", given, nil)

	// -- Then
	//
	a.EqualError(err, "the annotation struct must not be nil")
}

func (a *AnnosTestSuite) TestFromMapNilMap() {
	// -- Given
	//
	actual := new(Bools)
	expected := new(Bools)

	// -- When
	//
	err := FromMap("canary.kage.cloud", nil, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

func (a *AnnosTestSuite) TestFromMapNonStructPtr() {
	// -- Given
	//
	actual := map[string]string{}

	// -- When
	//
	err := FromMap("canary.kage.cloud", map[string]string{}, &actual)

	// -- Then
	//
	a.EqualError(err, "the annotation must be a ptr to a struct")
}

func (a *AnnosTestSuite) TestToMapEmbedded() {
	// -- Given
	//
	given := Embeded{
		Strings: Strings{
			String: "str1",
		},
		StringOuter: "str",
	}
	s := "ptr"
	given.StringPtr = &s
	expected := map[string]string{
		"StringOuter": "str",
		"string":      "str1",
		"string_ptr":  "ptr",
	}

	// -- When
	//
	actual := ToMap("", given)

	// -- Then
	//
	a.Equal(expected, actual)
}

func (a *AnnosTestSuite) TestFromMapEmbedded() {
	// -- Given
	//
	expected := &Embeded{
		Strings: Strings{
			String: "str1",
		},
		StringOuter: "str",
	}
	s := "ptr"
	expected.StringPtr = &s
	actual := new(Embeded)
	given := map[string]string{
		"StringOuter": "str",
		"string":      "str1",
		"string_ptr":  "ptr",
	}

	// -- When
	//
	err := FromMap("", given, actual)

	// -- Then
	//
	if a.NoError(err) {
		a.Equal(expected, actual)
	}
}

type Ints struct {
	Int       int     `json:"int"`
	Int8      int8    `json:"int_8"`
	Int16     int16   `json:"int_16"`
	Int32     int32   `json:"int_32"`
	Int64     int64   `json:"int_64"`
	UInt      uint    `json:"u_int"`
	UInt8     uint8   `json:"u_int_8"`
	UInt16    uint16  `json:"u_int_16"`
	UInt32    uint32  `json:"u_int_32"`
	UInt64    uint64  `json:"u_int_64"`
	IntPtr    *int    `json:"int_ptr"`
	Int8Ptr   *int8   `json:"int_8_ptr"`
	Int16Ptr  *int16  `json:"int_16_ptr"`
	Int32Ptr  *int32  `json:"int_32_ptr"`
	Int64Ptr  *int64  `json:"int_64_ptr"`
	UIntPtr   *uint   `json:"u_int_ptr"`
	UInt8Ptr  *uint8  `json:"u_int_8_ptr"`
	UInt16Ptr *uint16 `json:"u_int_16_ptr"`
	UInt32Ptr *uint32 `json:"u_int_32_ptr"`
	UInt64Ptr *uint64 `json:"u_int_64_ptr"`
}

type Floats struct {
	Float32    float32  `json:"float_32"`
	Float64    float64  `json:"float_64"`
	Float32Ptr *float32 `json:"float_32_ptr"`
	Float64Ptr *float64 `json:"float_64_ptr"`
}

type Strings struct {
	String    string  `json:"string"`
	StringPtr *string `json:"string_ptr,omitempty"`
}

type Bools struct {
	Bool    bool  `json:"bool"`
	BoolPtr *bool `json:"bool_ptr,omitempty"`
}

type Maps struct {
	MapStringString          map[string]string            `json:"map_string_string"`
	MapStringInt             map[string]int               `json:"map_string_int"`
	MapStringFloat32         map[string]float32           `json:"map_string_float_32"`
	MapStringFloat64         map[string]float64           `json:"map_string_float_64"`
	MapStringBool            map[string]bool              `json:"map_string_bool"`
	MapStringMapStringString map[string]map[string]string `json:"map_string_map_string_string"`
}

type Slices struct {
	Strings []string  `json:"strings"`
	Ints    []int     `json:"ints"`
	Floats  []float32 `json:"floats"`
	Bools   []bool    `json:"bools"`
}

type Nested struct {
	Bools     Bools             `json:"bools"`
	Strings   *Strings          `json:"strings"`
	Map       map[string]string `json:"map"`
	SubNested SubNested         `json:"sub_nested"`
}

type SubNested struct {
	Slices Slices `json:"slices"`
	hidden bool   `json:"hidden"`
}

type Embeded struct {
	Strings
	StringOuter string
}

func TestAnnosTestSuite(t *testing.T) {
	suite.Run(t, new(AnnosTestSuite))
}

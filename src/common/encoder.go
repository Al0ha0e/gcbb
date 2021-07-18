package common

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"io"
	"reflect"
)

type Encoder interface {
	Encode(interface{}) []byte
	Decode([]byte, interface{})
}

type NaiveNetEncoder struct{}

func encodeTypes(buf io.Writer, tp reflect.Type, val reflect.Value, tag string) {
	kind := tp.Kind()
	for ; kind == reflect.Ptr; kind = tp.Kind() {
		tp = tp.Elem()
		val = val.Elem()
	}
	switch kind {
	case reflect.Chan:
		return
	case reflect.Bool:
		binary.Write(buf, binary.BigEndian, val.Bool())
	case reflect.Int:
		binary.Write(buf, binary.BigEndian, val.Int())
	case reflect.Int8:
		binary.Write(buf, binary.BigEndian, int8(val.Int()))
	case reflect.Int16:
		binary.Write(buf, binary.BigEndian, int16(val.Int()))
	case reflect.Int32:
		binary.Write(buf, binary.BigEndian, int32(val.Int()))
	case reflect.Int64:
		binary.Write(buf, binary.BigEndian, val.Int())
	case reflect.Uint:
		binary.Write(buf, binary.BigEndian, val.Uint())
	case reflect.Uint8:
		binary.Write(buf, binary.BigEndian, uint8(val.Uint()))
	case reflect.Uint16:
		binary.Write(buf, binary.BigEndian, uint16(val.Uint()))
	case reflect.Uint32:
		binary.Write(buf, binary.BigEndian, uint32(val.Uint()))
	case reflect.Uint64:
		binary.Write(buf, binary.BigEndian, val.Uint())
	case reflect.Uintptr:
		binary.Write(buf, binary.BigEndian, val.Uint())
	case reflect.Float32:
		binary.Write(buf, binary.BigEndian, float32(val.Float()))
	case reflect.Float64:
		binary.Write(buf, binary.BigEndian, val.Float())
	case reflect.Array:
		for i := 0; i < val.Len(); i++ {
			encodeTypes(buf, tp.Elem(), val.Index(i), "")
		}
	case reflect.String:
		binary.Write(buf, binary.BigEndian, uint32(val.Len()))
		for i := 0; i < val.Len(); i++ {
			encodeTypes(buf, val.Index(i).Type(), val.Index(i), "")
		}
	case reflect.Slice:
		binary.Write(buf, binary.BigEndian, uint32(val.Len())) //MAY BUG
		for i := 0; i < val.Len(); i++ {
			encodeTypes(buf, tp.Elem(), val.Index(i), "")
		}
	case reflect.Struct:
		for i := 0; i < tp.NumField(); i++ {
			field := tp.Field(i)
			// fmt.Println(field.Type)
			encodeTypes(buf, field.Type, val.Field(i), string(field.Tag))
		}
	}
}

func decodeTypes(buf io.Reader, tp reflect.Type, val reflect.Value, tag string) {
	kind := tp.Kind()
	switch kind {
	case reflect.Ptr:
		val = val.Elem()
		decodeTypes(buf, tp.Elem(), val, "")
	case reflect.Chan:
		return
	case reflect.Bool:
		var v bool
		binary.Read(buf, binary.BigEndian, &v)
		val.SetBool(v)
	case reflect.Int:
		var v int64
		binary.Read(buf, binary.BigEndian, &v)
		val.SetInt(v)
	case reflect.Int8:
		var v int8
		binary.Read(buf, binary.BigEndian, &v)
		val.SetInt(int64(v))
	case reflect.Int16:
		var v int16
		binary.Read(buf, binary.BigEndian, &v)
		val.SetInt(int64(v))
	case reflect.Int32:
		var v int32
		binary.Read(buf, binary.BigEndian, &v)
		val.SetInt(int64(v))
	case reflect.Int64:
		var v int64
		binary.Read(buf, binary.BigEndian, &v)
		val.SetInt(int64(v))
	case reflect.Uint:
		var v uint64
		binary.Read(buf, binary.BigEndian, &v)
		val.SetUint(v)
	case reflect.Uint8:
		var v uint8
		binary.Read(buf, binary.BigEndian, &v)
		val.SetUint(uint64(v))
	case reflect.Uint16:
		var v uint16
		binary.Read(buf, binary.BigEndian, &v)
		val.SetUint(uint64(v))
	case reflect.Uint32:
		var v uint32
		binary.Read(buf, binary.BigEndian, &v)
		val.SetUint(uint64(v))
	case reflect.Uint64:
		var v uint64
		binary.Read(buf, binary.BigEndian, &v)
		val.SetUint(uint64(v))
	case reflect.Uintptr:
		var v uint
		binary.Read(buf, binary.BigEndian, &v)
		val.SetUint(uint64(v))
	case reflect.Float32:
		var v float32
		binary.Read(buf, binary.BigEndian, &v)
		val.SetFloat(float64(v))
	case reflect.Float64:
		var v float64
		binary.Read(buf, binary.BigEndian, &v)
		val.SetFloat(float64(v))
	case reflect.Array:
		for i := 0; i < val.Len(); i++ {
			decodeTypes(buf, tp.Elem(), val.Index(i), "")
		}
		//fmt.Println("VAL", val)
	case reflect.String:
		var l uint32
		binary.Read(buf, binary.BigEndian, &l)
		bts := make([]byte, l)
		for i := 0; i < int(l); i++ {
			var b uint8
			binary.Read(buf, binary.BigEndian, &b)
			bts[i] = b
		}
		s := string(bts)
		val.SetString(s)
	case reflect.Slice:
		var l uint32
		binary.Read(buf, binary.BigEndian, &l)
		//fmt.Println("LLL", l)
		for i := 0; i < int(l); i++ {
			nval := reflect.New(tp.Elem()).Elem()
			//fmt.Println("NVAL", nval)
			decodeTypes(buf, tp.Elem(), nval, "")
			//fmt.Println("NVAL", nval)
			val.Set(reflect.Append(val, nval))
		}

	case reflect.Struct:
		for i := 0; i < tp.NumField(); i++ {
			field := tp.Field(i)
			decodeTypes(buf, field.Type, val.Field(i), string(field.Tag))
			//fmt.Println("OK", i, val.Field(i))
		}
	}
}

func (nne *NaiveNetEncoder) Encode(val interface{}) []byte {
	buf := new(bytes.Buffer)

	t := reflect.TypeOf(val)
	v := reflect.ValueOf(val)
	encodeTypes(buf, t, v, "")
	return buf.Bytes()
}

func (nne *NaiveNetEncoder) Decode(data []byte, val interface{}) {
	buf := bytes.NewReader(data)
	vof := reflect.ValueOf(val)
	decodeTypes(buf, reflect.TypeOf(val), vof, "")
}

type GobNetEncoder struct{}

func (gne *GobNetEncoder) Encode(val interface{}) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(val)
	return buf.Bytes()
}
func (gne *GobNetEncoder) Decode(data []byte, val interface{}) {
	decoder := gob.NewDecoder(bytes.NewReader(data))
	decoder.Decode(val)
}

package libs

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

func Struct2Map(a interface{}) (map[string]any, error) {
	str, err := jsoniter.Marshal(a)
	if err != nil {
		return nil, err
	}
	var ret map[string]any
	err = jsoniter.Unmarshal(str, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func MySHA256(str string) string {
	tmp := sha256.New()
	tmp.Write([]byte(str))
	return fmt.Sprintf("%X", tmp.Sum(nil))
}

func RandomString(length int) string {
	const alpha string = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	str := strings.Builder{}
	for i := 0; i < length; i++ {
		str.WriteString(string(alpha[rand.Intn(62)]))
	}
	return str.String()
}

func Reverse[T any](arr []T) {
	l := len(arr)
	for i := 0; i*2 < l; i++ {
		arr[i], arr[l-i-1] = arr[l-i-1], arr[i]
	}
}

func HasInt(srt []int, val int) bool {
	i := sort.SearchInts(srt, val)
	return i < len(srt) && srt[i] == val
}

func HasElement[T comparable](arr []T, val T) bool {
	return FindSlice(arr, val) >= 0
}

func JoinArray[T any](val []T) string {
	s := strings.Builder{}
	for i, j := range val {
		s.WriteString(fmt.Sprint(j))
		if i+1 < len(val) {
			s.WriteString(",")
		}
	}
	return s.String()
}

func If[T any](a bool, b T, c T) T {
	if a {
		return b
	}
	return c
}

func GetTempDir() string {
	for {
		tmp := TmpDir + RandomString(16)
		_, err := os.Stat(tmp)
		if err != nil {
			os.MkdirAll(tmp, os.ModePerm)
			return tmp
		}
	}
}

func UnzipMemory(mem []byte) (map[string][]byte, error) {
	//OpenReader will open the Zip file specified by name and return a ReadCloser.
	reader, err := zip.NewReader(bytes.NewReader(mem), int64(len(mem)))
	if err != nil {
		return nil, err
	}
	ret := make(map[string][]byte)
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		w := bytes.NewBuffer(nil)
		_, err = io.Copy(w, rc)
		if err != nil {
			return nil, err
		}
		ret[file.Name] = w.Bytes()
		rc.Close()
	}
	return ret, nil
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func AtoiDefault(str string, def int) int {
	val, err := strconv.Atoi(str)
	if err != nil {
		return def
	}
	return val
}

/*
Whether a is start with b
*/
func StartsWith(a, b string) bool {
	if len(a) < len(b) {
		return false
	}
	return a[: len(b)] == b
}

func TimeStamp() int64 {
	return time.Now().UnixMilli()
}

func DeepCopy(dst, src any) error {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(src)
	if err != nil {
		return err
	}
	return gob.NewDecoder(&buf).Decode(dst)
}

type Numbers interface {
	int | int8 | int16 | int32 | int64 | float32 | float64 | uint | uint8 | uint16 | uint32 | uint64
}

func Max[T Numbers](a, b T) T {
	if a < b {
		return b
	} else {
		return a
	}
}

func Min[T Numbers](a, b T) T {
	if a < b {
		return a
	} else {
		return b
	}
}

func FindSlice[T comparable](a []T, val T) int {
	for i := range a {
		if a[i] == val {
			return i
		}
	}
	return -1
}

func DeleteSlice[T any](a []T, id int) {
	a = append(a[:id], a[id+1:]...)
}
/*
resort an sorted array after one entry has modified
*/
func ResortEntry[T any](a []T, f func(int, int) bool, id int) {
	t := a[id]
	for id > 0 && !f(id - 1, id) {
		a[id] = a[id - 1]
		a[id - 1] = t
	}
	for id + 1 < len(a) && !f(id, id + 1) {
		a[id] = a[id + 1]
		a[id + 1] = t
	}
}
package libs

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
)

func Struct2Map(a interface{}) (map[string]any, error) {
	str, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	var ret map[string]any
	err = json.Unmarshal(str, &ret)
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

func HasIntN(arr []int, val int) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
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

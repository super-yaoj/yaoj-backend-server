package libs

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
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
	for i := 0; i * 2 < l; i++ {
		arr[i], arr[l - i - 1] = arr[l - i - 1], arr[i]
	}
}

func HasInt(srt []int, val int) bool {
	i := sort.SearchInts(srt, val)
	return i < len(srt) && srt[i] == val
}

func HasIntN(arr []int, val int) bool {
	for _, v := range arr {
		if v == val { return true }
	}
	return false
}

func JoinArray[T any](val []T) string {
	s := strings.Builder{}
	for i, j := range val {
		s.WriteString(fmt.Sprint(j))
		if i + 1 < len(val) {
			s.WriteString(",")
		}
	}
	return s.String()
}

func If[T any](a bool, b T, c T) T {
	if a { return b }
	return c
}

func GetTempDir() string {
	for {
		tmp := TmpDir + RandomString(16)
		_, err := os.Stat(tmp)
		if err != nil {
			os.Mkdir(tmp, os.ModePerm)
			return tmp
		}
	}
}

func ZipDir(src_dir string, zip_file_name string) error {
	dir, err := ioutil.ReadDir(src_dir)
	if err != nil {
		return err
	}
	if len(dir) == 0 {
		return  nil
	}
	os.RemoveAll(zip_file_name)
	zipfile, _ := os.Create(zip_file_name)
	defer zipfile.Close()
	archive := zip.NewWriter(zipfile)
	defer archive.Close()
 
	filepath.Walk(src_dir, func(path string, info os.FileInfo, _ error) error {
		if path == src_dir {
			return nil
		}
		header, _ := zip.FileInfoHeader(info)
 
		header.Name = path[len(src_dir) + 1 :]
		if info.IsDir() {
			header.Name += `/`
		} else {
			header.Method = zip.Deflate
		}
 
		writer, _ := archive.CreateHeader(header)
		if !info.IsDir() {
			file, _ := os.Open(path)
			defer file.Close()
			io.Copy(writer, file)
		}
		return nil
	})
	return  nil
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func AtoiDefault(str string, def int) int {
	val, err := strconv.Atoi(str)
	if err != nil { return def }
	return val
}
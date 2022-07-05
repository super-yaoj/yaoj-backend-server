package libs

import (
	"os"
	"time"
)

var (
	FrontDomain  string      = "http://localhost:8080"
	BackDomain   string      = "http://localhost:8081"
	Sault        string      = "3.1y4a1o5j9"
	DefaultGroup int         = 1
	Day          int         = 86400
	Month        int         = Day * 30
	Year         int         = Day * 365
	DataDir      string      = "local/data"
	TmpDir       string      = "local/tmp"
	CacheMap     MemoryCache = NewMemoryCache(time.Hour, 1000)
	LangSuf      []string    = []string{".cpp", ".cpp11.cpp", ".cpp14.cpp", ".cpp20.cpp", ".py2.py", ".py3.py", ".go", ".java", ".c"}
)

func init() {
	os.MkdirAll(DataDir, os.ModePerm)
	os.MkdirAll(TmpDir, os.ModePerm)
}

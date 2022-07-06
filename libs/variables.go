package libs

import (
	"os"
	"time"
)

var (
	FrontDomain  string
	BackDomain   string
	DataDir      string
	TmpDir       string
	Sault        string      = "3.1y4a1o5j9"
	DefaultGroup int         = 1
	Day          int         = 86400
	Month        int         = Day * 30
	Year         int         = Day * 365
	CacheMap     MemoryCache = NewMemoryCache(time.Hour, 1000)
	LangSuf      []string    = []string{".cpp", ".cpp11.cpp", ".cpp14.cpp", ".cpp20.cpp", ".py2.py", ".py3.py", ".go", ".java", ".c"}
)

func DirInit() {
	os.MkdirAll(DataDir, os.ModePerm)
	os.MkdirAll(TmpDir, os.ModePerm)
}

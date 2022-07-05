package libs

import "time"

var (
	FrontDomain  string        = "http://www.wzyhome.work"
	BackDomain   string        = "http://localhost:8081"
	Sault        string        = "3.1y4a1o5j9"
	DefaultGroup int           = 1
	Day          int           = 86400
	Month        int           = Day * 30
	Year         int           = Day * 365
	DataDir      string        = "D:/Work/codes/web/yao/data/"
	TmpDir		 string 	   = "D:/Work/codes/web/yao/tmp/"
	CacheMap     MemoryCache   = NewMemoryCache(time.Hour, 1000)
	LangSuf		 []string 	   = []string{".cpp", ".cpp11.cpp", ".cpp14.cpp", ".cpp20.cpp", ".py2.py", ".py3.py", ".go", ".java", ".c"}
)
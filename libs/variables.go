package libs

import (
	"os"
)

var (
	FrontDomain  string
	BackDomain   string
	DataDir      string
	TmpDir       string
	Sault        string   = "3.1y4a1o5j9"
	DefaultGroup int      = 1
	Day          int      = 86400
	Month        int      = Day * 30
	Year         int      = Day * 365
	LangSuf      []string = []string{".cpp98.cpp", ".cpp11.cpp", ".cpp14.cpp", ".cpp17.cpp", ".cpp20.cpp", ".py2.py", ".py3.py", ".go", ".java", ".c"}
	DataSource   string
)

func DirInit() {
	os.MkdirAll(DataDir, os.ModePerm)
	os.MkdirAll(TmpDir, os.ModePerm)
}

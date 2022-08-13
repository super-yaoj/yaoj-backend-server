package main

import (
	"fmt"
	"strings"
	"text/template"
	"time"
)

var tsIntfTpl = template.Must(template.New("tsintf").Parse(
	`{ {{range $key, $value := .}}{{ $key }}: {{ $value }}; {{end -}} }`))

var tsFuncTpl = template.Must(template.New("tsdecl").Parse(
	`export function call(path: {{.Path}}, method: {{.Method}}{{if .HasOption}}, option: { {{if .ParamType}}param: {{.ParamType}};{{end}} {{if .BodyType}}body: {{.BodyType}};{{end}} }{{end}}): Promise<any>;`))

type FuncDeclData struct {
	Path      string
	Method    string
	ParamType string
	BodyType  string
	HasOption bool
}

func ParseType(fields []ParamField) string {
	obj := map[string]string{}

	for _, pf := range fields {
		switch pf.typ {
		case "int", "*int":
			obj[pf.name] = "int"
		case "string", "*string":
			obj[pf.name] = "string"
		case "*multipart.FileHeader":
			obj[pf.name] = "any"
		default:
			panic(fmt.Sprintf("unknown type %q", pf.typ))
		}
	}
	// pp.Print(obj)

	var buf strings.Builder
	tsIntfTpl.Execute(&buf, obj)
	s := buf.String()
	if s == "{ }" {
		s = ""
	}
	return s
}

func RenderTsDecl() {
	fmt.Printf("// generate by routeparse at %v\n", time.Now())
	for pth, api := range apis {
		for _, item := range api {
			var buf strings.Builder
			sq, sb := ParseType(params[item.handler].query), ParseType(params[item.handler].body)
			tsFuncTpl.Execute(&buf, FuncDeclData{
				Path:      pth,
				Method:    item.method,
				ParamType: sq,
				BodyType:  sb,
				HasOption: sq != "" || sb != "",
			})
			s := buf.String()
			fmt.Println(s)
		}
	}
}

package main

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"path/filepath"
	"regexp"
	"strings"
)

type Api struct {
	handler string
	method  string
}

var apis = map[string][]Api{}

type KeyValueVisitor struct {
	path string // route path
}

func (v *KeyValueVisitor) Visit(node ast.Node) ast.Visitor {
	switch decl := node.(type) {
	case *ast.CompositeLit:
		if len(decl.Elts) == 2 {
			if basiclit, ok := decl.Elts[0].(*ast.BasicLit); ok {
				callexpr := decl.Elts[1].(*ast.CallExpr)

				if apis[v.path] == nil {
					apis[v.path] = make([]Api, 0)
				}
				apis[v.path] = append(apis[v.path], Api{callexpr.Args[0].(*ast.Ident).Name, basiclit.Value})
				return nil
			}
		}
	}
	return v
}

type RouterVisitor struct {
}

func (v *RouterVisitor) Visit(node ast.Node) ast.Visitor {
	switch decl := node.(type) {
	case *ast.KeyValueExpr:
		path := decl.Key.(*ast.BasicLit).Value
		// fmt.Println(path)
		return &KeyValueVisitor{path: path}
	}
	return v
}

// Visitor
type RouterSeeker struct {
}

func (v *RouterSeeker) Visit(node ast.Node) ast.Visitor {
	switch decl := node.(type) {
	case *ast.ValueSpec:
		if len(decl.Names) == 1 && decl.Names[0].Name == "Router" {
			return &RouterVisitor{}
		}
		return nil
	case *ast.FuncDecl:
		return nil
	case *ast.ImportSpec:
		return nil
	}
	return v
}

type ParamField struct {
	name string
	typ  string
	tags []string // split validation tag by ','
}

type Param struct {
	hasAuth bool
	query   []ParamField
	body    []ParamField
	doc     string
}

var params = map[string]*Param{}

type ParamVisitor struct {
	name string
	doc  string
}

var tagParse = regexp.MustCompile(`^(.*?):"(.*?)"$`)

func (v *ParamVisitor) parseTag(tag string, typename string) {
	tag = tag[1 : len(tag)-1] // remove side `
	tags := strings.Split(tag, " ")

	var origin string
	var field ParamField
	field.typ = typename

	for _, token := range tags {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		match := tagParse.FindSubmatch([]byte(token))
		if len(match) == 3 {
			key, val := string(match[1]), string(match[2])
			log.Println(key, val, typename)
			switch key {
			case "query", "body":
				origin = key
				field.name = val
			case "validate":
				field.tags = strings.Split(val, ",")
			}
		}
	}
	if origin == "query" {
		params[v.name].query = append(params[v.name].query, field)
	} else if origin == "body" {
		params[v.name].body = append(params[v.name].body, field)
	}
}

func (v *ParamVisitor) Visit(node ast.Node) ast.Visitor {
	switch decl := node.(type) {
	case *ast.Field:
		// pp.Print(decl)
		if decl.Tag != nil {
			var buf strings.Builder
			format.Node(&buf, fset, decl.Type)
			v.parseTag(decl.Tag.Value, buf.String())
			// pp.Print(decl.Type)
		}
		if ident, ok := decl.Type.(*ast.Ident); ok {
			switch ident.Name {
			case "Auth":
				params[v.name].hasAuth = true
			case "Page":
				params[v.name].query = append(params[v.name].query, ParamField{
					name: "left",
					typ:  "*int",
				}, ParamField{
					name: "right",
					typ:  "*int",
				}, ParamField{
					name: "pagesize",
					typ:  "*int",
					tags: []string{"gte=1", "lte=100"},
				})
			}
			//pp.Print(field)
		}
	case *ast.StructType:
		if params[v.name] == nil {
			params[v.name] = &Param{}
		}
		params[v.name].doc = v.doc
	}
	return v
}

type ParamSeeker struct {
	withComment string // comments of the nearest GenDecl
}

var endWithParam = regexp.MustCompile(`Param$`)

func (v *ParamSeeker) Visit(node ast.Node) ast.Visitor {

	switch decl := node.(type) {
	case *ast.GenDecl:
		v.withComment = ""
		if decl.Doc != nil {
			v.withComment = decl.Doc.Text()
		}
		return v
	case *ast.TypeSpec:
		if endWithParam.Match([]byte(decl.Name.Name)) {
			// remove Param suffix
			return &ParamVisitor{
				name: decl.Name.Name[:len(decl.Name.Name)-5],
				doc:  v.withComment,
			}
		}
		return nil
	case *ast.FuncDecl:
		return nil
	case *ast.ImportSpec:
		return nil
	}
	return v
}

var fset = token.NewFileSet()

func main() {
	// 这里取绝对路径，方便打印出来的语法树可以转跳到编辑器
	path, _ := filepath.Abs("./controllers")
	pkgs, err := parser.ParseDir(fset, path, nil, parser.AllErrors|parser.ParseComments)
	f := pkgs["controllers"]
	if err != nil {
		log.Println(err)
		return
	}
	ast.Walk(&RouterSeeker{}, f)
	// pp.Print(apis)
	ast.Walk(&ParamSeeker{}, f)
	// pp.Print(params)
	// ast.Print(fset, f)
	RenderTsDecl()
}

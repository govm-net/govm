package acb2fb3994c274446f5dd4d8397d2f73ad68f32f649e2577c23877f3a4d7e1a05

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/lengzhao/govm/counter"
	"github.com/lengzhao/govm/runtime"
)

// CreateAppFromSourceCode create app
func CreateAppFromSourceCode(fileName string, flag byte) ([]byte, uint64) {
	info := newAppInfo{}
	l := counter.Annotate(fileName, "")
	info.LineNum = uint32(l)
	info.Flag = flag

	var depends []DependItem
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatalf("fail to read file: %s: %s", fileName, err)
	}

	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "", content, parser.ParseComments)
	if err != nil {
		log.Fatal("fail to parse code:", err)
	}
	for _, s := range f.Imports {
		item := DependItem{}
		data := getDepName(unquote(s.Path.Value))
		runtime.Decode(data, &item.AppName)
		if len(s.Name.Name) > 4 {
			log.Fatalln("the alias of import too long(<=4):", s.Name.Name)
		}
		for i, v := range []byte(s.Name.Name) {
			item.Alias[i] = v
		}
		depends = append(depends, item)
		//log.Println("depend:", unquote(s.Path.Value), s.End())
	}
	if len(depends) > 254{
		log.Fatalf("too many depends: %d", len(depends))
	}
	info.DependNum = uint8(len(depends))

	//log.Println("decl:", len(f.Decls))
	var vis visitor
	vis.imports = make([]int, 0)
	ast.Walk(&vis, f)
	lst := f.Decls
	f.Decls = nil
	for i, v := range lst {
		found := false
		for _, j := range vis.imports {
			if i == j {
				found = true
			}
		}
		if found {
			continue
		}
		f.Decls = append(f.Decls, v)
	}

	buff := bytes.NewBuffer([]byte{})
	printer.Fprint(buff, fs, f)

	out := runtime.Encode(info)
	for _, dep := range depends {
		name := runtime.Encode(dep)
		out = append(out, name...)
	}
	code, _ := ioutil.ReadAll(buff)
	r := regexp.MustCompile("^package.*\n")
	code = r.ReplaceAll(code, []byte{})
	r = regexp.MustCompile("\npackage.*\n")
	code = r.ReplaceAll(code, []byte{10})

	r = regexp.MustCompile("^// .build.*\n")
	code = r.ReplaceAll(code, []byte{})

	if flag&AppFlagGzipCompress > 0 {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, err := zw.Write(code)
		if err != nil {
			log.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			log.Fatal(err)
		}
		code = buf.Bytes()
	}

	out = append(out, code...)
	appName := runtime.GetHash(out)
	log.Printf("code info,fileName:%s,info:%v,appName:%x\n",fileName, info, appName)
	return out, l
}

func getDepName(depend string) []byte {
	split := strings.Split(depend, "/")
	name := split[len(split)-1]
	appName, _ := hex.DecodeString(name[1:])
	if len(appName) != HashLen {
		log.Fatalln("error depend:", depend, name, len(appName))
	}
	//log.Println("depend:", depend, name, appName)
	return appName
}

// unquote returns the unquoted string.
func unquote(s string) string {
	t, err := strconv.Unquote(s)
	if err != nil {
		log.Fatalf("cover: improperly quoted string %q\n", s)
	}
	return t
}

type visitor struct {
	index   int
	imports []int
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.GoStmt:
		panic(n)
	case *ast.GenDecl:
		v.index++
		if n.Tok != token.IMPORT && n.Tok != token.PACKAGE {
			return v
		}
		v.imports = append(v.imports, v.index-1)
	case *ast.Package:
		log.Println("package:", v.index, n)
	default:
	}
	return v
}

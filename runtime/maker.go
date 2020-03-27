package runtime

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"text/template"
	"time"

	"github.com/lengzhao/govm/conf"
	"github.com/lengzhao/govm/counter"
)

// TDependItem app的依赖信息
type TDependItem struct {
	Alias   [4]byte
	AppName [32]byte
}

// TAppNewHead 新建app的头消息，不包含依赖列表
type TAppNewHead struct {
	LineNum   uint32
	Type      uint16
	Flag      uint8
	DependNum uint8
}

// TAppNewInfo 新建app的信息，不包含依赖列表
type TAppNewInfo struct {
	TAppNewHead
	Depends []TDependItem
}

const (
	// AppFlagRun the app can be call
	AppFlagRun = uint8(1 << iota)
	// AppFlagImport the app code can be included
	AppFlagImport
	// AppFlagPlublc App funds address uses the plublc address, except for app, others have no right to operate the address.
	AppFlagPlublc
	// AppFlagGzipCompress gzip compress
	AppFlagGzipCompress
	// AppFlagEnd end of flag
	AppFlagEnd
)

// var envItems = []string{"GO111MODULE=on"}
var envItems = []string{}

const execName = "app.exe"
const codeName = "app.go"

func init() {
	os.RemoveAll(path.Join(projectRoot, "app_main"))
}

// NewApp 创建app
func NewApp(chain uint64, name []byte, code []byte) {
	//1.生成原始文件，go build，校验是否正常
	//2.添加代码统计
	//3.如果可执行，添加执行代码
	c := conf.GetConf()
	if bytes.Compare(c.CorePackName, name) == 0 {
		filePath := GetFullPathOfApp(chain, name)
		dstFileName := path.Join(filePath, "core.go")
		os.RemoveAll(filePath)
		createDir(filePath)
		s1, err := template.ParseFiles("./core/core.tmpl")
		if err != nil {
			log.Println("fail to ParseFiles core.tmpl:", err)
			panic(err)
		}
		f, err := os.Create(dstFileName)
		if err != nil {
			log.Println("fail to create run file:", dstFileName, err)
			panic(err)
		}
		packPath := GetPackPath(chain, name)
		info := TAppInfo{hexToPackageName(name), packPath, filePath, chain}
		err = s1.Execute(f, info)
		if err != nil {
			log.Println("fail to execute run file:", dstFileName, err)
			f.Close()
			panic(err)
		}
		f.Close()
		return
	}

	nInfo := TAppNewInfo{}
	n := Decode(code, &nInfo.TAppNewHead)
	assert(nInfo.Type == 0)
	if nInfo.Flag >= AppFlagEnd {
		panic("error flag")
	}
	if NotRebuild {
		var needBuild = false
		filePath := GetFullPathOfApp(chain, name)
		if nInfo.Flag&AppFlagImport != 0 {
			codeFile := path.Join(BuildDir, filePath, codeName)
			_, err := os.Stat(codeFile)
			if os.IsNotExist(err) {
				needBuild = true
			}
		}
		if nInfo.Flag&AppFlagRun != 0 {
			binFile := path.Join(BuildDir, filePath, execName)
			_, err := os.Stat(binFile)
			if os.IsNotExist(err) {
				needBuild = true
			}
		}

		if !needBuild {
			return
		}
	}

	code = code[n:]
	for i := 0; i < int(nInfo.DependNum); i++ {
		item := TDependItem{}
		n := Decode(code, &item)
		code = code[n:]
		nInfo.Depends = append(nInfo.Depends, item)
	}
	if nInfo.Flag&AppFlagGzipCompress != 0 {
		buf := bytes.NewBuffer(code)
		var out bytes.Buffer
		zr, err := gzip.NewReader(buf)
		if err != nil {
			log.Fatal("gzip.NewReader", err)
		}
		if _, err := io.Copy(&out, zr); err != nil {
			log.Fatal("io.Copy", err)
		}

		if err := zr.Close(); err != nil {
			log.Fatal("zr.Close()", err)
		}
		code, err = ioutil.ReadAll(&out)
		if err != nil {
			log.Fatal("ioutil.ReadAll", err)
		}
	}

	appName := hexToPackageName(name)
	filePath := GetFullPathOfApp(chain, name)
	dstFileName := path.Join(filePath, codeName)
	srcFilePath := path.Join(projectRoot, "temp", fmt.Sprintf("chain%d", chain), codeName)

	dstRelFN := path.Join(BuildDir, dstFileName)
	srcRelFN := path.Join(BuildDir, srcFilePath)
	createDir(path.Dir(srcRelFN))
	createDir(path.Dir(dstRelFN))
	// defer os.RemoveAll("temp")

	//判断源码是否已经存在，如果存在，则直接执行，返回有效代码行数
	//生成原始代码文件
	f, err := os.Create(srcRelFN)
	if err != nil {
		log.Println("fail to create go file:", srcRelFN, err)
		panic(err)
	}
	createSourceFile(chain, appName, nInfo.Depends, code, f)
	f.Close()
	//编译、校验原始代码
	cmd := exec.Command("go", "build", srcFilePath)
	cmd.Dir = BuildDir
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	cmd.Env = os.Environ()
	for _, item := range envItems {
		cmd.Env = append(cmd.Env, item)
	}

	err = cmd.Run()
	if err != nil {
		log.Println("fail to build source file:", srcFilePath, err)
		panic(err)
	}

	//为原始代码添加代码统计，生成目标带统计的代码文件
	lineNum := counter.Annotate(srcRelFN, dstRelFN)
	if lineNum != uint64(nInfo.LineNum) {
		log.Println("error line number:", lineNum, ",hope:", nInfo.LineNum)
		panic(lineNum)
	}

	//再次编译，确认没有代码冲突
	cmd = exec.Command("go", "build", dstFileName)
	cmd.Dir = BuildDir
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	cmd.Env = os.Environ()
	for _, item := range envItems {
		cmd.Env = append(cmd.Env, item)
	}
	err = cmd.Run()
	if err != nil {
		log.Println("fail to build source file:", srcFilePath, err)
		panic(err)
	}

	if nInfo.Flag&AppFlagRun != 0 {
		makeAppExe(chain, name)
	}
	// os.RemoveAll("temp")
}

func createSourceFile(chain uint64, packName string, depends []TDependItem, code []byte, w io.Writer) {
	if bytes.Index(code, []byte("import")) != -1 {
		panic("code include 'import'")
	}

	if bytes.Index(code, []byte("_consume_tip_")) != -1 {
		panic("code include '_consume_tip_'")
	}

	w.Write([]byte("package "))
	w.Write([]byte(packName))
	w.Write([]byte("\n\n"))

	for _, item := range depends {
		realName := GetPackPath(chain, item.AppName[:])
		w.Write([]byte("import "))
		w.Write(item.Alias[:])
		w.Write([]byte(" \""))
		w.Write([]byte(realName))
		w.Write([]byte("\"\n"))
	}

	r := regexp.MustCompile("^// .build.*\n")
	code = r.ReplaceAll(code, []byte{})

	w.Write(code)
}

func hexToPackageName(in []byte) string {
	return "a" + hex.EncodeToString(in)
}

// TAppInfo app info
type TAppInfo struct {
	AppName  string
	PackPath string
	CorePath string
	ChainID  uint64
}

func makeAppExe(chain uint64, name []byte) {
	//1.add func GoVMRun
	//2.make func main
	//3.build
	//4.delete func GoVMRun
	c := conf.GetConf()
	packPath := GetPackPath(chain, name)
	corePath := GetPackPath(chain, c.CorePackName)
	info := TAppInfo{hexToPackageName(name), packPath, corePath, chain}
	s1, err := template.ParseFiles(path.Join(BuildDir, "run.tmpl"))
	if err != nil {
		log.Println("fail to ParseFiles run.tmpl:", err)
		panic(err)
	}
	realPath := GetFullPathOfApp(chain, name)
	runFile := path.Join(BuildDir, realPath, "run.go")
	defer os.Remove(runFile)
	f, err := os.Create(runFile)
	if err != nil {
		log.Println("fail to create run file:", runFile, err)
		panic(err)
	}
	err = s1.Execute(f, info)
	if err != nil {
		log.Println("fail to execute run file:", runFile, err)
		f.Close()
		panic(err)
	}
	f.Close()
	// log.Println("create fun file:", runFile)
	fn := path.Join(projectRoot, "app_main", fmt.Sprintf("chain%d_%d", chain, time.Now().UnixNano()), "main.go")
	relFN := path.Join(BuildDir, fn)

	exeFile := path.Join(path.Dir(fn), execName)
	relDir := path.Dir(relFN)
	createDir(relDir)
	defer os.RemoveAll(relDir)

	fm, _ := os.Create(relFN)
	defer fm.Close()
	s2, _ := template.ParseFiles(path.Join(BuildDir, "main.tmpl"))
	s2.Execute(fm, info)

	//再次编译，确认没有代码冲突
	cmd := exec.Command("go", "build", "-o", exeFile, fn)
	cmd.Dir = BuildDir
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	cmd.Env = os.Environ()
	for _, item := range envItems {
		cmd.Env = append(cmd.Env, item)
	}
	err = cmd.Run()
	if err != nil {
		log.Println("fail to build source file:", exeFile, fn, err)
		panic(err)
	}
	binFile := path.Join(BuildDir, realPath, execName)
	os.Remove(binFile)
	os.Rename(path.Join(BuildDir, exeFile), binFile)
}

// RebuildApp rebuild app
func RebuildApp(chain uint64, dir string) error {
	err := filepath.Walk(dir, func(fPath string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if f.Name() != execName {
			return nil
		}
		dir := filepath.Dir(fPath)
		name := filepath.Base(dir)
		if name == "" {
			log.Println("unknow path:", fPath)
			return nil
		}
		appName, err := hex.DecodeString(name[1:])
		if err != nil {
			log.Println("fail to decode:", name, err)
			return nil
		}
		makeAppExe(chain, appName)
		return nil
	})
	if err != nil {
		log.Println("fail:", err)
	}
	return err
}

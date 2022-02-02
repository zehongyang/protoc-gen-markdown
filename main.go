package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ditashi/jsbeautifier-go/jsbeautifier"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func main() {
	g := markdown{rmp: make(map[string]Router)}
  	var flags flag.FlagSet
  	flags.StringVar(&g.Router, "router", "routers","router sets")
	protogen.Options{ParamFunc: flags.Set}.Run(g.Generate)
}


type Router struct {
	Path string `json:"path"`
	Method string `json:"method"`
	Comment string `json:"comment"`
	Query string `json:"query"`
}

type markdown struct {
	Prefix string
	Router string
	msgs []protoreflect.FullName
	rmp map[string]Router
}

func (md *markdown) in(m *protogen.Message) {
	md.msgs = append(md.msgs, m.Desc.FullName())

}

func (md *markdown) out() {
	md.msgs = md.msgs[0 : len(md.msgs)-1]
}

func (md *markdown) recursive(m *protogen.Message) bool {
	for _, n := range md.msgs {
		if n == m.Desc.FullName() {
			return true
		}
	}
	return false
}

var letters = []string{"-","=","|"}
func dealComment (comment string) string {
	for _, letter := range letters {
		comment = strings.ReplaceAll(comment,letter,"")
	}
	return comment
}

func (md *markdown) Generate(plugin *protogen.Plugin) error {
	// The service should be defined in the last file.
	// All other files are imported by the service proto.
	err := md.rangeRouterTxts()
	if err != nil {
		return err
	}
	f := plugin.Files[len(plugin.Files)-1]
	//if len(f.Services) == 0 {
	//	return nil
	//}
	var initFlag bool
	fname := *f.Proto.Name + ".md"
	t := plugin.NewGeneratedFile(fname, "")
	t.P("# ","Protocol Documentation")
	t.P("<a name=\"top\"></a>")
	t.P("## ","Table of Contents")
	t.P("- ","[",*f.Proto.Name,"]","(","#",*f.Proto.Name,")")

	for _, enum := range f.Enums {
		initFlag = true
		t.P("\t","- ","[",enum.Desc.Name(),"]","(","#",enum.Desc.Name(),")")
	}
	for _, msg := range f.Messages {
		initFlag = true
		t.P("\t","- ","[",msg.Desc.Name(),"]","(","#",msg.Desc.Name(),")")
	}
	if !initFlag {
		return nil
	}
	var sb strings.Builder
	for _, enum := range f.Enums {
		t.P(fmt.Sprintf("<a name=\"%s\"></a>",enum.Desc.Name()))
		t.P("### ",enum.Desc.Name())
		for _, comments := range enum.Comments.LeadingDetached {
			//t.P(dealComment(strings.TrimRight(string(comments),"\r\n")))
			t.P(strings.TrimRight(string(comments),"\r\n"))
		}
		t.P(strings.TrimRight(string(enum.Comments.Leading),"\r\n"))
		t.P()
		t.P("| ","Name","| ","Number","| ","Description")
		t.P("| ---- | ------ | ----------- |")
		for _, value := range enum.Values {
			sb.WriteString("| ")
			sb.WriteString(string(value.Desc.Name()))
			sb.WriteString("| ")
			sb.WriteString(fmt.Sprintf("%d",int(value.Desc.Number())))
			sb.WriteString("| ")
			//sld := strings.TrimRight(string(value.Comments.Leading),"\r\n")
			//if len(sld) > 0 {
			//	sb.WriteString(sld)
			//}
			stl := strings.TrimRight(string(value.Comments.Trailing),"\r\n")
			stl = strings.Trim(stl,"-")
			if len(stl) > 0 {
				//if len(sld) > 0 {
				//	sb.WriteString(";")
				//}
				sb.WriteString(stl)
			}
			sb.WriteString("|")
			t.P(sb.String())
			sb.Reset()
		}
		sb.Reset()
	}

	for _, msg := range f.Messages {
		t.P(fmt.Sprintf("<a name=\"%s\"></a>",msg.Desc.Name()))
		t.P("### ", msg.Desc.Name())
		t.P()
		for _, comments := range msg.Comments.LeadingDetached {
			//t.P("说明：",dealComment(strings.TrimRight(string(comments),"\r\n")))
			t.P("说明：",strings.TrimRight(string(comments),"\r\n"))
		}
		t.P(strings.TrimRight(string(msg.Comments.Leading),"\r\n"))
		t.P()
		t.P("| ","Name","| ","Type","| ","Array","| ","Description")
		t.P("| ---- | ---- | ----- | ----------- |")
		for _, field := range msg.Fields {
			sb.WriteString("| ")
			sb.WriteString(string(field.Desc.Name()))
			sb.WriteString("| ")
			if field.Message != nil {
				var kd = field.Message.Desc.Name()
				if field.Desc.IsMap() {
					kd = "Map"
				}
				sb.WriteString(fmt.Sprintf("[%s](%s.md#%s)",kd,
					field.Message.Location.SourceFile,kd))
			}else if field.Enum != nil {
				sb.WriteString(fmt.Sprintf("[%s](%s.md#%s)",field.Enum.Desc.Name(),
					field.Enum.Location.SourceFile,field.Enum.Desc.Name()))
			}else{
				var kd = field.Desc.Kind().String()
				sb.WriteString(fmt.Sprintf("[%s](%s.md#%s)",kd,
					field.Location.SourceFile,kd))
			}
			sb.WriteString("| ")
			if field.Desc.IsList() {
				sb.WriteString("Yes")
			}else{
				sb.WriteString("No")
			}
			sb.WriteString("| ")
			//sld := strings.TrimRight(string(field.Comments.Leading),"\r\n")
			//if len(sld) > 0 {
			//	sb.WriteString(sld)
			//}
			stl := dealComment(strings.TrimRight(string(field.Comments.Trailing),"\r\n"))
			if len(stl) > 0 {
				//if len(sld) > 0 {
				//	sb.WriteString(";")
				//}
				sb.WriteString(stl)
			}
			sb.WriteString("|")
			t.P(sb.String())
			sb.Reset()
		}
		r,ok := md.rmp[string(msg.Desc.Name())]
		if ok {
			t.P("```")
			t.P(r.Path," ",strings.Trim(r.Method,"#"))
			t.P("```")
		}
		t.P("```json")
		t.P(md.jsDocForMessage(msg))
		t.P("```")
	}

	t.P()
	//content, _ := t.Content()
	//tfn.Write(content)
	return nil
}

func (md *markdown) rangeRouterTxts () error {
	if len(md.Router) < 1 {
		return nil
	}
	fis, err := ioutil.ReadDir(md.Router)
	if err != nil {
		return nil
	}
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}
		if fi.Name() == "." || fi.Name() == ".." {
			continue
		}
		err = md.parseRouterTxt(filepath.Join(md.Router, fi.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}

func (md *markdown) parseRouterTxt (fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()
	rf := bufio.NewReader(f)
	for  {
		ld, _, err := rf.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if len(ld) > 0 {
			var rt Router
			ss := strings.Split(string(ld), "\t")
			if len(ss) < 1 {
				continue
			}
			rt.Path = ss[0]
			if len(ss) > 2 {
				rt.Method = ss[2]
			}
			if len(ss) > 3 {
				rt.Query = ss[3]
			}
			if len(ss) > 4 {
				rt.Comment = ss[4]
			}
			md.rmp[rt.Query] = rt
		}
	}
	return nil
}



func (md *markdown) api(s string) string {
	i := strings.LastIndex(s, ".")

	prefix := strings.Trim(md.Prefix, "/")
	if prefix != "" {
		prefix = "/" + prefix
	}

	return prefix + "/" + s[:i] + "/" + s[i+1:]
}

func (md *markdown) anchor(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "/", "")
	return s
}

func (md *markdown) scalarDefaultValue(field *protogen.Field) string {
	switch field.Desc.Kind() {
	case protoreflect.StringKind, protoreflect.BytesKind:
		return `""`
	case protoreflect.Fixed64Kind, protoreflect.Int64Kind,
		protoreflect.Sfixed64Kind, protoreflect.Sint64Kind,
		protoreflect.Uint64Kind:
		return `0`
	case protoreflect.DoubleKind, protoreflect.FloatKind:
		return `0.0`
	case protoreflect.BoolKind:
		return `false`
	default:
		return "0"
	}
}

func (md *markdown) jsDocForField(field *protogen.Field) string {
	//js := field.Comments.Leading.String()
	js := ""
	js += fmt.Sprintf("\"%s\"",string(field.Desc.Name())) + ":"

	var vv string
	var vt string
	if field.Desc.IsMap() {
		vf := field.Message.Fields[1]
		if m := vf.Message; m != nil {
			vv = md.jsDocForMessage(m)
			vt = string(vf.Message.Desc.FullName())
		} else {
			vv = md.scalarDefaultValue(vf)
			vt = vf.Desc.Kind().String()
		}
		kf := field.Desc.MapKey()
		vv = fmt.Sprintf("{}")
		vt = fmt.Sprintf("%s,%s", kf.Kind().String(), vt)
	} else if field.Message != nil {
		if md.recursive(field.Message) {
			vv = "{}"
		} else {
			vv = md.jsDocForMessage(field.Message)
		}
		vt = string(field.Message.Desc.Name())
	} else if field.Enum != nil {
		//vv = `"` + string(field.Enum.Values[0].Desc.Name()) + `"`
		vv = `0`
		vt = string(field.Enum.Desc.Name())
		//for i, v := range field.Enum.Values {
		//	if i > 0 {
		//		vt += ","
		//	}
		//	vt += string(v.Desc.Name())
		//}
	} else if field.Oneof != nil {
		vv = `"Does Not Support OneOf"`
	} else {
		vv = md.scalarDefaultValue(field)
		vt = field.Desc.Kind().String()
	}

	if field.Desc.IsList() {
		js += fmt.Sprintf("[%s],", vv)
	} else if field.Desc.IsMap() {
		js += vv + fmt.Sprintf(", // map<%s>", vt)
	} else if field.Enum != nil {
		js += vv + fmt.Sprintf(", ")
	} else {
		js += vv + fmt.Sprintf(", ")
	}

	if t := string(field.Comments.Trailing); len(t) > 0 {
		//js += ", " + strings.TrimLeft(t, " ")
	} else {
	}
	js += "\n"

	return js
}

func (md *markdown) jsDocForMessage(m *protogen.Message) string {
	md.in(m)
	defer md.out()

	js := "{\n"

	for _, field := range m.Fields {
		js += md.jsDocForField(field)
	}

	js += "}"
	options := jsbeautifier.DefaultOptions()
	js, _ = jsbeautifier.Beautify(&js, options)

	return js
}

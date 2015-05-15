package main

import (
    "bytes"
    "database/sql"
    "flag"
    "fmt"
    _ "github.com/go-sql-driver/mysql"
    "log"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    tt "text/template"
)

const (
    dsn          = "test:test@tcp(test:3306)/demo?timeout=3s&parseTime=true&loc=Local&charset=utf8"
    outPut       = "./dao"
    suffix       = ".go"
    templateFile = "./template"
)

var (
    err      error
    db       *sql.DB
    template *tt.Template
    funcMap  = make(tt.FuncMap)
)

type Desc struct {
    Field string
    Type  string
    Null  string
    Key   string
}

func init() {
    db, err = sql.Open("mysql", dsn)
    if err != nil {
        panic(err)
    }
    if err = db.Ping(); err != nil {
        panic(err)
    }

    // mkdir outPut dir
    os.Mkdir(outPut, 0755)

    // template
    funcMap["fill"] = fill
    funcMap["genNumStr"] = genNumStr
    template, err = tt.New(filepath.Base(templateFile)).Funcs(funcMap).ParseFiles(templateFile)
    if err != nil {
        log.Printf("new template error(%v)", err)
    }
}

func main() {
    flag.Parse()

    // show tables
    cmd := "show tables"
    rows, err := db.Query(cmd)
    if err != nil {
        log.Printf("db.Exec(%v) error(%v)", cmd, err)
        return
    }

    // model files
    files := make(map[string]*os.File)

    // write code
    for rows.Next() {
        // get tables
        var name string
        rows.Scan(&name)
        files[name], err = os.OpenFile(filepath.Join(outPut, name+suffix), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
        if err != nil {
            log.Printf("os.OpenFile(%v) error(%v)", name+suffix, err)
        }

        // get table members
        cmd = fmt.Sprintf("desc `%s`", name)
        r, err := db.Query(cmd)
        if err != nil {
            log.Printf("db.Query(%v) error(%v)", cmd, err)
            continue
        }

        // template
        data := make(map[string]interface{})
        fields := make([]map[string]string, 0)
        fieldPriNames := make([]string, 0)
        fieldNames := make([]string, 0)
        FieldPriNames := make([]string, 0)
        FieldNames := make([]string, 0)
        for r.Next() {
            desc := Desc{}
            r.Scan(&desc.Field, &desc.Type, &desc.Null, &desc.Key, nil, nil)
            dField := transferName(desc.Field)
            dType := transferType(desc.Type)
            if desc.Key == "PRI" {
                data["priName"] = desc.Field
                data["PriName"] = transferName(dField)
                data["priType"] = dType
            } else {
                fieldNames = append(fieldNames, desc.Field)
                FieldNames = append(FieldNames, dField)
            }
            fieldPriNames = append(fieldPriNames, desc.Field)
            FieldPriNames = append(FieldPriNames, dField)
            fields = append(fields, map[string]string{"fName": dField, "fType": dType, "sep": "\n"})
        }
        data["name"] = name
        data["Name"] = transferName(name)
        data["fields"] = fields
        data["fieldPriNames"] = fieldPriNames
        data["FieldPriNames"] = FieldPriNames
        data["fieldNames"] = fieldNames
        data["FieldNames"] = FieldNames
        data["count"] = len(fieldNames)
        data["priCount"] = len(FieldPriNames)
        err = template.Execute(files[name], data)
        if err != nil {
            log.Printf("template.Execute() error(%v)", err)
        }
    }

    // write models
    for _, file := range files {
        file.Close()
    }
}

func transferName(str string) string {
    runes := make([]rune, 0)
    toUpper := false
    l := len(str)
    for i, v := range str {
        if toUpper && v >= 97 && v <= 122 {
            v -= 32
            toUpper = false
            runes = append(runes, v)
        } else {
            runes = append(runes, v)
        }
        if i < l && v == 95 {
            toUpper = true
        }
    }
    return strings.Title(strings.Replace(string(runes), "_", "", -1))
}

func transferType(str string) string {
    mc := regexp.MustCompile(`\w+`)
    t := mc.FindStringSubmatch(str)[0]
    s := ""
    switch t {
        case "tinyint", "int":
        s = "int"
        case "bigint":
        s = "int64"
        case "float", "decimal":
        s = "float32"
        case "double":
        s = "float64"
        case "timestamp", "date", "datetime":
        s = "time.Time"
        case "binary", "varbinary":
        s = "[]byte"
        case "char", "varchar":
        s = "string"
        default:
        s = "string"
    }
    return s
}

func fill(a []string, prefix, suffix string) string {
    length := len(a)
    s := make([]string, length)
    buffer := bytes.NewBufferString("")
    for i := 0; i < len(a); i++ {
        if prefix != "" {
            buffer.WriteString(prefix)
        }
        buffer.WriteString(a[i])
        if suffix != "" {
            buffer.WriteString(suffix)
        }
        s[i] = buffer.String()
        buffer.Reset()
    }
    return strings.Join(s, ", ")
}

func genNumStr(num int, str string, sep string) string {
    l := make([]string, num)
    for i := 0; i < num; i++ {
        l[i] = str
    }
    return strings.Join(l, sep)
}

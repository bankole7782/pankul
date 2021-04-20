package pankul

import (
  "strings"
  "net/http"
  "fmt"
  "strconv"
  "os"
  "html/template"
  "math/rand"
  "time"
  "runtime"
)

func getBaseTemplate() string {
  if BaseTemplate != "" {
    return BaseTemplate
  } else {
    return "pankul_files/bad-base.html"
  }
}


func errorPage(w http.ResponseWriter, msg string) {
  _, fn, line, _ := runtime.Caller(1)
  type Context struct {
    Message template.HTML
    SourceFn string
    SourceLine int
    PANDOLEE_DEVELOPER bool
  }

  var ctx Context
  if os.Getenv("PANDOLEE_DEVELOPER") == "true" {
    msg = strings.ReplaceAll(msg, "\n", "<br>")
    msg = strings.ReplaceAll(msg, "\t", "&nbsp;&nbsp;&nbsp;")
    ctx = Context{template.HTML(msg), fn, line, true}
  } else {
    ctx = Context{template.HTML(msg), fn, line, false}
  }
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "pankul_files/error-page.html"))
  tmpl.Execute(w, ctx)
}


func isUserAdmin(r *http.Request) (bool, error) {
  userid, err := GetCurrentUser(r)
  if err != nil {
    return false, err
  }
  for _, id := range Admins {
    if userid == id {
      return true, nil
    }
  }
  return false, nil
}


func GetDocumentStructureList() ([]string, error) {
  tempSlice := make([]string, 0)

  rows, err := FRCL.Search(`
  	table: pk_document_structures
		fields: fullname
  	`)
	if err != nil {
		return tempSlice, err
	}

  for _, row := range *rows {
  	tempSlice = append(tempSlice, row["fullname"].(string))
  }

  return tempSlice, nil
}


func untestedRandomString(length int) string {
  var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
  const charset = "abcdefghijklmnopqrstuvwxyz1234567890"

  b := make([]byte, length)
  for i := range b {
    b[i] = charset[seededRand.Intn(len(charset))]
  }
  return string(b)
}


func newTableName() (string, error) {
  for {
    newName := "pktbl_" + untestedRandomString(3)
    count, err := FRCL.CountRows(fmt.Sprintf(`
    	table: pk_document_structures
    	where:
    		tbl_name = %s
    	`, newName))
    if err != nil {
      return "", err
    }
    if count == 0 {
      return newName, nil
    }
  }
}


func tableName(documentStructure string) (string, error) {
	row, err := FRCL.SearchForOne(fmt.Sprintf(`
		table: pk_document_structures
		fields: tbl_name
		where:
			fullname = '%s'
		`, documentStructure))
	if err != nil {
		return "", err
	}
	return (*row)["tbl_name"].(string), nil
}


func optionSearch(commaSeperatedOptions, option string) bool {
  if commaSeperatedOptions == "" {
    return false
  } else {
    options := strings.Split(commaSeperatedOptions, ",")
    optionsTrimmed := make([]string, 0)
    for _, opt := range options {
      optionsTrimmed = append(optionsTrimmed, strings.TrimSpace(opt))
    }
    for _, value := range optionsTrimmed {
      if option == value {
        return true
      }
    }
    return false
  }
}


func docExists(documentName string) (bool, error) {
  dsList, err := GetDocumentStructureList()
  if err != nil {
    return false, err
  }

  for _, value := range dsList {
    if value == documentName {
      return true, nil
    }
  }
  return false, nil
}


func getDocumentStructureID(documentStructure string) (int64, error) {
	row, err := FRCL.SearchForOne(fmt.Sprintf(`
		table: pk_document_structures
		where:
			fullname = '%s'
		`, documentStructure))
	if err != nil {
		return 0, err
	}

  idStr := (*row)["id"].(string)
  idInt64, err := strconv.ParseInt(idStr, 10, 64)
  if err != nil {
    return 0, err
  }
	return idInt64, nil
}


type DocData struct {
  Label string
  Name string
  Type string
  Required bool
  Unique bool
  OtherOptions []string
}


func GetDocData(documentStructure string) ([]DocData, error) {
  dds := make([]DocData, 0)
  dsid, err := getDocumentStructureID(documentStructure)
  if err != nil {
    return dds, err
  }

  rows, err := FRCL.Search(fmt.Sprintf(`
  	table: pk_fields
  	order_by: view_order asc
  	where:
  		dsid = %d
  	`, dsid))
  if err != nil {
    return dds, err
  }
  for _, row := range *rows {
    var label, name, type_, options, otherOptions string

    label = row["label"].(string)
    name = row["name"].(string)
    type_ = row["type"].(string)
    if op, ok := row["options"]; ok {
    	options = op.(string)
    }
    if oo, ok := row["other_options"]; ok {
    	otherOptions = oo.(string)
    }
    var required, unique bool
    if optionSearch(options, "required") {
      required = true
    }
    if optionSearch(options, "unique") {
      unique = true
    }
    otherOptionsOk := make([]string, 0)
    for _, otherOption := range strings.Split(otherOptions, "\n") {
      otherOptionsOk = append(otherOptionsOk, strings.TrimSpace(otherOption))
    }
    dd := DocData{label, name, type_, required, unique, otherOptionsOk}
    dds = append(dds, dd)
  }

  return dds, nil
}


func BoolToStr(b bool) string {
  if b {
    return "t"
  } else {
    return "f"
  }
}


func getUserTimeZoneSuffix(userid int64) (string, error) {
  row, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: users
    where:
      id = %d
    `, userid))
  if err != nil {
    return "", err
  }

  loc, err := time.LoadLocation((*row)["timezone"].(string))
  if err != nil {
    return "", err
  }
  tzname, _ := time.Now().In(loc).Zone()
  return tzname, nil
}


func timeInUserTimeZone(t time.Time, userid int64) (time.Time, error) {
  row, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: users
    where:
      id = %d
    `, userid))
  if err != nil {
    return time.Time{}, err
  }

  loc, err := time.LoadLocation((*row)["timezone"].(string))
  if err != nil {
    return time.Time{}, err
  }
  if err == nil {
    t = t.In(loc)
  }
  return t, nil
}

func cleanComment(rawComment string) string {
	return strings.ReplaceAll(rawComment, "\n", "<br>")
}

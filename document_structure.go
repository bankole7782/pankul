package pankul

import (
  "net/http"
  "fmt"
  "strconv"
  "strings"
  "html/template"
  "github.com/gorilla/mux"
)


func newDocumentStructure(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  if r.Method == http.MethodGet {

    type Context struct {
      DocumentStructures string
    }
    dsList, err := GetDocumentStructureList()
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    ctx := Context{strings.Join(dsList, ",,,")}

    tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "pankul_files/new-document-structure.html"))
    tmpl.Execute(w, ctx)

  } else {
    type QFField struct {
      label string
      name string
      type_ string
      options string
      other_options string
    }

    qffs := make([]QFField, 0)
    r.ParseForm()
    for i := 1; i < 100; i++ {
      iStr := strconv.Itoa(i)
      if r.FormValue("label-" + iStr) == "" {
        break
      } else {
        qff := QFField{
          label: r.FormValue("label-" + iStr),
          name: r.FormValue("name-" + iStr),
          type_: r.FormValue("type-" + iStr),
          options: strings.Join(r.PostForm["options-" + iStr], ","),
          other_options: r.FormValue("other-options-" + iStr),
        }
        qffs = append(qffs, qff)
      }
    }

    tblName, err := newTableName()
    if err != nil {
      errorPage(w, err.Error())
      return
    }
    toInsert := map[string]interface{} {
      "fullname": r.FormValue("ds-name"),
      "tbl_name": tblName,
    }
    if len(strings.TrimSpace(r.FormValue("comment"))) != 0 {
      toInsert["comment"] = r.FormValue("comment")
    }

    dsid, err := FRCL.InsertRowAny("pk_document_structures", toInsert)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    for i, o := range(qffs) {
      toInsertQFFields := map[string]interface{} {
        "dsid": dsid, "label": o.label, "name": o.name, "type": o.type_, "options": o.options,
        "other_options": o.other_options, "view_order": i + 1,
      }
      _, err = FRCL.InsertRowAny("pk_fields", toInsertQFFields)
      if err != nil {
        errorPage(w, err.Error())
        return
      }
    }

    // create actual form data tables, we've only stored the form structure to the database
    stmt := fmt.Sprintf(`
    table: %s
    fields:
      created datetime required
      modified datetime required
      created_by int required
    `, tblName)

    stmtEnding := ""

    for _, qff := range qffs {
      if qff.type_ == "Section Break" {
        continue
      }
      stmt += "\n" + qff.name + " "
      if qff.type_ == "Check" {
        stmt += "bool"
      } else if qff.type_ == "Date" {
        stmt += "date"
      } else if qff.type_ == "Datetime" {
        stmt += "datetime"
      } else if qff.type_ == "Float" {
        stmt += "float"
      } else if qff.type_ == "Int" {
        stmt += "int"
      } else if qff.type_ == "Link" {
        stmt += "int"
      } else if qff.type_ == "Data" || qff.type_ == "Email" || qff.type_ == "URL" || qff.type_ == "Select" {
        stmt += "string"
      } else if qff.type_ == "Text" {
        stmt += "text"
      }
      if optionSearch(qff.options, "required") {
        stmt += " required"
      }

      if optionSearch(qff.options, "unique") {
        stmt += " unique"
      }

      if qff.type_ == "Link" {
        ottblName, err := tableName(qff.other_options)
        if err != nil {
          errorPage(w, err.Error())
          return
        }
        stmtEnding += fmt.Sprintf("\n%s %s on_delete_delete", qff.name, ottblName)
      }
    }
    stmtEnding += "\ncreated_by users on_delete_delete"

    stmt += "\n::\nforeign_keys:" + stmtEnding + "\n::"

    err = FRCL.CreateTable(stmt)
    if err != nil {
      errorPage(w, err.Error())
      return
    }

    http.Redirect(w, r, "/pk/list-document-structures/", 307)
  }

}


func listDocumentStructures(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  type DS struct{
    DSName string
  }

  structDSList := make([]DS, 0)

  rows, err := FRCL.Search(`
    table: pk_document_structures
    fields: fullname
    order_by: fullname asc
    `)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  for _, row := range *rows {
    structDSList = append(structDSList, DS{row["fullname"].(string)})
  }

  type Context struct {
    DocumentStructures []DS
  }

  ctx := Context{DocumentStructures: structDSList}

  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "pankul_files/list-document-structures.html"))
  tmpl.Execute(w, ctx)
}


func deleteDocumentStructure(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  vars := mux.Vars(r)
  ds := vars["document-structure"]

  detv, err := docExists(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if detv == false {
    errorPage(w, fmt.Sprintf("The document structure %s does not exists.", ds))
    return
  }

  tblName, err := tableName(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = FRCL.DeleteTable(tblName)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = FRCL.DeleteRows(fmt.Sprintf(`
    table: pk_document_structures
    where:
      fullname = '%s'
    `, ds))
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  var redirectURL string
  redirectURL = "/pk/list-document-structures/"
  http.Redirect(w, r, redirectURL, 307)
}


func viewDocumentStructure(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  vars := mux.Vars(r)
  ds := vars["document-structure"]

  detv, err := docExists(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if detv == false {
    errorPage(w, fmt.Sprintf("The document structure %s does not exists.", ds))
    return
  }

  row, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: pk_document_structures
    where:
      fullname = '%s'
    `, ds))
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  idStr := (*row)["id"].(string)
  id, err := strconv.ParseInt(idStr, 10, 64)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  tblName := (*row)["tbl_name"].(string)
  var commentStr string
  if htAny, ok := (*row)["comment"]; ok {
    commentStr = htAny.(string)
  }

  docDatas, err := GetDocData(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  type Context struct {
    DocumentStructure string
    DocDatas []DocData
    Id int64
    Add func(x, y int) int
    TableName string
    Comment template.HTML
  }

  add := func(x, y int) int {
    return x + y
  }

  ctx := Context{ds, docDatas, id, add, tblName, template.HTML(cleanComment(commentStr))}
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "pankul_files/view-document-structure.html"))
  tmpl.Execute(w, ctx)
}



func newDSFromTemplate(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  vars := mux.Vars(r)
  ds := vars["document-structure"]

  detv, err := docExists(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if detv == false {
    errorPage(w, fmt.Sprintf("The document structure %s does not exists.", ds))
    return
  }

  docDatas, err := GetDocData(ds)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  row, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: pk_document_structures
    where:
      fullname = '%s'
    `, ds))
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  var commentStr string
  if htAny, ok := (*row)["comment"]; ok {
    commentStr = htAny.(string)
  }

  dsList, err := GetDocumentStructureList()
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  type Context struct {
    DocDatas []DocData
    DocumentStructure string
    Add func(x, y int) int
    Comment string
    FormatOtherOptions func([]string) string
    DocumentStructures string
  }

  add := func(x, y int) int {
    return x + y
  }

  ffunc := func(x []string) string {
    return strings.Join(x, "\n")
  }

  ctx := Context{docDatas, ds, add, commentStr, ffunc, strings.Join(dsList, ",,,")}
  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "pankul_files/new-ds-from-template.html"))
  tmpl.Execute(w, ctx)
}

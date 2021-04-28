package pankul

import (
  "github.com/pkg/errors"
  "net/http"
  "fmt"
  "github.com/bankole7782/flaarum"
  "time"
  "strconv"
  "html"
  "strings"
)




type DocData struct {
  Label string
  Name string
  Type string
  Required bool
  Unique bool
  OtherOptions []string
}


// This function can be used for rendering a form.
// You are to loop through the returned object to create a form.
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


// Gets a dataMap (map[string]string) from the request object.
func MakeDataMapFromRequest(r *http.Request) map[string]string {
  outMap := make(map[string]string)

  r.FormValue("email") // this would populate the r.PostForm object.
  for k, _ := range r.PostForm {
    outMap[k] = r.FormValue(k)
  }

  return outMap
}


// This function would be used for saving in your create pages.
func CreateDocument(ds string, dataMap map[string]string, r *http.Request) (int64, error) {
  userIdInt64, err := GetCurrentUser(r)
  if err != nil {
    return 0, errors.Wrap(err, "pankul error")
  }

  detv, err := docExists(ds)
  if err != nil {
    return 0, errors.Wrap(err, "pankul error")
  }
  if detv == false {
    return 0, errors.New(fmt.Sprintf("The document structure %s does not exists.", ds))
  }

  dds, err := GetDocData(ds)
  if err != nil {
    return 0, errors.Wrap(err, "pankul error")
  }

  tblName, err := tableName(ds)
  if err != nil {
    return 0, errors.Wrap(err, "pankul error")
  }
  toInsert := make(map[string]string)
  for _, dd := range dds {
    if dd.Type == "Section Break" {
      continue
    }
    tmpData, ok := dataMap[dd.Name]
    if ! ok {
      continue
    }
    switch dd.Type {
    case "Check":
      var data string
      if tmpData == "on" {
        data = "t"
      } else {
        data = "f"
      }
      toInsert[dd.Name] = data

    case "Datetime":
      if tmpData != "" {
        tzname, err := getUserTimeZoneSuffix(userIdInt64)
        if err != nil {
          return 0, errors.Wrap(err, "pankul error")
        }
        toInsert[dd.Name] = tmpData + " " + tzname
      }

    default:
      if tmpData != "" {
        toInsert[dd.Name] = tmpData
      }
    }
  }

  toInsert["created"] = flaarum.RightDateTimeFormat(time.Now())
  toInsert["modified"] = flaarum.RightDateTimeFormat(time.Now())
  toInsert["created_by"] = fmt.Sprintf("%d", userIdInt64)

  lastIdStr, err := FRCL.InsertRowStr(tblName, toInsert)
  if err != nil {
    return 0, errors.Wrap(err, "pankul error")
  }

  lastId, err := strconv.ParseInt(lastIdStr, 10, 64)
  if err != nil {
    return 0, errors.Wrap(err, "strconv error")
  }

  return lastId, nil
}


type DocAndStructure struct {
  DocData
  Data string
}

// This function can be used in your update pages or view pages.
// This function would get the structure and data for presentation as a form with some values.
// The second field contains meta information: created, created_by & modified.
func GetDocument(ds string, docId int64, r *http.Request) ([]DocAndStructure, map[string]string, error) {
  userIdInt64, err := GetCurrentUser(r)
  if err != nil {
    return nil, nil, errors.Wrap(err, "pankul error")
  }

  detv, err := docExists(ds)
  if err != nil {
    return nil, nil, errors.Wrap(err, "pankul error")
  }
  if detv == false {
    return nil, nil, errors.New(fmt.Sprintf("The document structure %s does not exists.", ds))
  }

  tblName, err := tableName(ds)
  if err != nil {
    return nil, nil, errors.Wrap(err, "pankul error")
  }

  count, err := FRCL.CountRows(fmt.Sprintf(`
    table: %s
    where:
      id = %d
    `, tblName, docId))
  if err != nil {
    return nil, nil, errors.Wrap(err, "flaarum error")
  }
  if count == 0 {
    return nil, nil, errors.New(fmt.Sprintf("The document with id %d do not exists", docId))
  }

  arow, err := FRCL.SearchForOne(fmt.Sprintf(`
    table: %s expand
    where:
      id = %d
    `, tblName, docId))
  if err != nil {
    return nil, nil, errors.Wrap(err, "flaarum error")
  }

  docDatas, err := GetDocData(ds)
  if err != nil {
    return nil, nil, errors.Wrap(err, "pankul error")
  }

  docAndStructureSlice := make([]DocAndStructure, 0)

  rowMap := make(map[string]string)
  for k, v := range *arow {
    var data string
    switch dInType := v.(type) {
    case int64, float64:
      data = fmt.Sprintf("%v", dInType)
    case time.Time:
      dInTypeCorrected, err := timeInUserTimeZone(dInType, userIdInt64)
      if err != nil {
        return nil, nil, errors.Wrap(err, "pankul error")
      }
      data = dInTypeCorrected.Format("2006-01-02T15:04")
    case string:
      data = dInType
    case bool:
      data = boolToStr(dInType)
    }

    rowMap[k] = data
  }


  for _, docData := range docDatas {
    if docData.Type == "Date" {
      data := (*arow)[docData.Name].(time.Time).Format("2006-01-02")
      docAndStructureSlice = append(docAndStructureSlice, DocAndStructure{docData, data})
    } else {
      data := rowMap[ docData.Name ]

      docAndStructureSlice = append(docAndStructureSlice, DocAndStructure{docData, data})
    }
  }

  meta := make(map[string]string)
  rawCreated := (*arow)["created"].(time.Time)
  createdCorrected, err := timeInUserTimeZone(rawCreated, userIdInt64)
  if err != nil {
    return nil, nil, errors.Wrap(err, "pankul error")
  }
  meta["created"] = flaarum.RightDateTimeFormat(createdCorrected)

  rawModified := (*arow)["modified"].(time.Time)
  modifiedCorrected, err := timeInUserTimeZone(rawModified, userIdInt64)
  if err != nil {
    return nil, nil, errors.Wrap(err, "pankul error")
  }
  meta["modified"] = flaarum.RightDateTimeFormat(modifiedCorrected)

  created_by := (*arow)["created_by"].(int64)
  meta["created_by"] = strconv.FormatInt(created_by, 10)

  return docAndStructureSlice, meta, nil
}


// This is used in your update pages.
// It should be called to complete an update action.
func UpdateDocument(ds string, docId int64, dataMap map[string]string, r *http.Request) error {
  userIdInt64, err := GetCurrentUser(r)
  if err != nil {
    return errors.Wrap(err, "pankul error")
  }

  detv, err := docExists(ds)
  if err != nil {
    return errors.Wrap(err, "pankul error")
  }
  if detv == false {
    return errors.New(fmt.Sprintf("The document structure %s does not exists.", ds))
  }

  tblName, err := tableName(ds)
  if err != nil {
    return errors.Wrap(err, "pankul error")
  }

  docAndStructureSlice, _, err := GetDocument(ds, docId, r)
  if err != nil {
    return errors.Wrap(err, "pankul error")
  }

  toUpdate := make(map[string]string)
  for _, docAndStructure := range docAndStructureSlice {
    tmpData, ok := dataMap[docAndStructure.DocData.Name]
    if ! ok {
      continue
    }
    if docAndStructure.Data != html.EscapeString(r.FormValue(docAndStructure.DocData.Name)) {
      switch docAndStructure.DocData.Type {
      case "Check":
        var data string
        if tmpData == "on" {
          data = "t"
        } else {
          data = "f"
        }
        toUpdate[docAndStructure.DocData.Name] = data
      case "Datetime":
        tzname, err := getUserTimeZoneSuffix(userIdInt64)
        if err != nil {
          return errors.Wrap(err, "pankul error")
        }
        toUpdate[docAndStructure.DocData.Name] = tmpData + " " + tzname
      default:
        toUpdate[docAndStructure.DocData.Name] = html.EscapeString(tmpData)
      }
    }

  }


  toUpdate["modified"] = flaarum.RightDateTimeFormat(time.Now())

  err = FRCL.UpdateRowsStr(fmt.Sprintf(`
    table: %s
    where:
      id = %d
    `, tblName, docId), toUpdate)
  if err != nil {
    return errors.Wrap(err, "pankul error")
  }

  return nil
}

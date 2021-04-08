package pankul

import (
  "github.com/pkg/errors"
  "net/http"
  "fmt"
  "github.com/bankole7782/flaarum"
  "time"
  "strconv"
)


func CreateDocument(ds string, r *http.Request) (int64, error) {
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
    switch dd.Type {
    case "Check":
      var data string
      if r.FormValue(dd.Name) == "on" {
        data = "t"
      } else {
        data = "f"
      }
      toInsert[dd.Name] = data

    case "Datetime":
      if r.FormValue(dd.Name) != "" {
        tzname, err := getUserTimeZoneSuffix(userIdInt64)
        if err != nil {
          return 0, errors.Wrap(err, "pankul error")
        }
        toInsert[dd.Name] = r.FormValue(dd.Name) + " " + tzname
      }

    default:
      if r.FormValue(dd.Name) != "" {
        toInsert[dd.Name] = r.FormValue(dd.Name)
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


type docAndStructure struct {
  DocData
  Data string
}


func GetDocument(ds string, docId int64, r *http.Request) ([]docAndStructure, map[string]string, error) {
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

  docAndStructureSlice := make([]docAndStructure, 0)

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
      data = BoolToStr(dInType)
    }

    rowMap[k] = data
  }


  for _, docData := range docDatas {
    if docData.Type == "Section Break" {
      docAndStructureSlice = append(docAndStructureSlice, docAndStructure{docData, ""})
    } else if docData.Type == "Date" {
      data := (*arow)[docData.Name].(time.Time).Format("2006-01-02")
      docAndStructureSlice = append(docAndStructureSlice, docAndStructure{docData, data})
    } else {
      data := rowMap[ docData.Name ]

      docAndStructureSlice = append(docAndStructureSlice, docAndStructure{docData, data})
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

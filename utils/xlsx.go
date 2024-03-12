package utils

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/tealeg/xlsx"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

type Xlsx struct {
	error error
}

func (x *Xlsx) Error() error {
	return x.error
}

// OpenBinary 读取二进制文件
func (x *Xlsx) OpenBinary(body []byte) (xFile *xlsx.File) {
	// 坑：tealeg/xlsx 不支持 .xls 格式的excel文件 https://github.com/tealeg/xlsx/issues/320
	xFile, err := xlsx.OpenBinary(body)
	if err != nil {
		x.error = errors.New(fmt.Sprintf("打开excel二进制文件失败, error: %s", err.Error()))
	}
	return
}

// OpenFile 读取文件
func (x *Xlsx) OpenFile(filename string) (xFile *xlsx.File) {
	xFile, err := xlsx.OpenFile(filename)
	if err != nil {
		x.error = errors.New(fmt.Sprintf("打开excel文件失败, error: %s", err.Error()))
	}
	return
}

// 读取sheet内容, 返回数组
func (x *Xlsx) readSheet(sheet *xlsx.Sheet) (result [][]string) {
	if x.error != nil {
		return
	}
	dataList := make([][]string, 0)
	// 循环每一行
	for _, row := range sheet.Rows {
		rowList := make([]string, 0)
		// 循环每一列
		for _, cell := range row.Cells {
			rowList = append(rowList, strings.Trim(cell.Value, " "))
		}
		dataList = append(dataList, rowList)
	}
	result = dataList
	return
}

// ReadSheetWithIndex 根据下标读取单个sheet数据
func (x *Xlsx) ReadSheetWithIndex(file *xlsx.File, sheetIndex int64) (result [][]string) {
	if x.error != nil {
		return
	}
	sheet := file.Sheets[sheetIndex]
	result = x.readSheet(sheet)
	return
}

// ReadSheetWithName 根据sheet名称读取单个sheet数据
func (x *Xlsx) ReadSheetWithName(file *xlsx.File, sheetName string) (result [][]string) {
	if x.error != nil {
		return
	}
	for _, sheet := range file.Sheets {
		if sheet.Name == sheetName {
			result = x.readSheet(sheet)
			return
		}
	}
	x.error = errors.New(fmt.Sprintf("表格sheet不存在, sheet_name: %s", sheetName))
	return
}

// ReadFile 读取excel文件，写入对应的titleMap对应的字段返回
func (x *Xlsx) ReadFile(filename string, titleMap map[string]string, dataIndex int64) (result []map[string]interface{}, err error) {
	/*
		filename: 文件绝对路径
		titleMap: title标题对应字段名称  {"名称": "name", "类型": "type"}
		dataIndex: 开始有数据的行
		result: 返回数据列表， [{"name": "test", "type"}]
		err: 错误信息
	*/
	xfile := x.OpenFile(filename)
	sheet := x.ReadSheetWithIndex(xfile, dataIndex)

	// 读取第一行的title值，从cmdb设置s好的title对应id map中获取相应的id，加入到TitleList，若未找到ID，则说明字段错误
	fieldList := make([]map[string]interface{}, 0) // [{"field": "name", "index": 0}]
	titleRow := sheet[0]
	// 取出表格第一行的title，根据title从CMDB信息中获取相应的字段名，加入到列表中
	for index, value := range titleRow {
		// 若值为空，则跳过
		value := strings.Trim(value, " ")
		if value == "" {
			continue
		}
		id := titleMap[value]
		if id == "" {
			zap.L().Info(fmt.Sprintf("值不存在： ", value))
			break
		}
		field := map[string]interface{}{"field": id, "index": index} // 此种实现，是防止有空列在中间存在
		fieldList = append(fieldList, field)
	}
	result = []map[string]interface{}{}
	// 循环后面的数据，将数据读入列表中
	for _, col := range sheet[1:] {
		// 循环整理好的fieldList,将对应的index加入到对应的值里
		data := map[string]interface{}{}
		for _, field := range fieldList {
			// 取出当前的字段和对应的下标信息
			fieldName := field["field"].(string)
			index := field["index"].(int)
			// 取出表格当前行对应的数据，若数据为空则跳过
			value := strings.Trim(col[index], " ")
			if value == "" {
				continue
			}
			// 将数据添加到data中
			data[fieldName] = value
		}
		result = append(result, data)
	}
	return
}

func (x *Xlsx) NewFile(titleFieldList []map[string]string, dataList []map[string]interface{}) (result *xlsx.File) {
	/*
		filename: 文件绝对路径
		titleFieldList: 标题 key map list	[{"title": "名称", "key": "name"}]
		dataList: 数据列表
	*/
	file := xlsx.NewFile()
	sheet, err := file.AddSheet("sheet1")
	if err != nil {
		return
	}
	// 添加标题
	row := sheet.AddRow()
	for _, item := range titleFieldList {
		// 添加第一条数据前，先添加field标题字段
		cell := row.AddCell()
		cell.Value = item["title"]

	}
	// 添加数据
	for _, data := range dataList {
		row := sheet.AddRow()
		for _, field := range titleFieldList {
			cell := row.AddCell()
			if value, ok := data[field["key"]]; ok {
				switch value.(type) {
				case string:
					cell.Value = value.(string)
				case float64:
					cell.Value = strconv.Itoa(int(value.(float64)))
				case int:
					cell.Value = strconv.Itoa(value.(int))
				}
			} else {
				cell.Value = ""
			}
		}
	}
	return file
}

func (x *Xlsx) SaveFile(filename string, titleFieldList []map[string]string, dataList []map[string]interface{}) {
	f := x.NewFile(titleFieldList, dataList)
	if f == nil {
		return
	}
	if err := f.Save(filename); err != nil {
		x.error = fmt.Errorf("保存数据到<%s>失败: <%s>", filename, err.Error())
	}
}

func (x *Xlsx) NewFileToBuffer(titleFieldList []map[string]string, dataList []map[string]interface{}) (result *bytes.Buffer) {
	f := x.NewFile(titleFieldList, dataList)
	if f == nil {
		return
	}
	buffer := new(bytes.Buffer)
	if err := f.Write(buffer); err != nil {
		x.error = fmt.Errorf("写入数据到buffer异常: <%s>", err.Error())
		return
	}
	return buffer
}

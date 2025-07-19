package godtemplate

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/beevik/etree"
)

type Replacer struct{}

type TableEntryStyle struct {
	CellStyle string
	TextStyle string
}

// Open the .odt/.zip file
func (r *Replacer) OpenFile(fileName string) (*zip.ReadCloser, error) {
	return zip.OpenReader(fileName)
}

// Read and parse content.xml
func (r *Replacer) GetDocument(zipReader *zip.ReadCloser) (*etree.Document, *zip.File, error) {
	var contentFile *zip.File
	for _, f := range zipReader.File {
		fmt.Println(f.Name)
		if f.Name == "content.xml" {
			contentFile = f
			break
		}
	}
	if contentFile == nil {
		return nil, nil, fmt.Errorf("content.xml not found")
	}

	rc, err := contentFile.Open()
	if err != nil {
		return nil, nil, err
	}
	defer rc.Close()

	buf := new(bytes.Buffer)
	io.Copy(buf, rc)

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(buf.Bytes()); err != nil {
		return nil, nil, err
	}
	return doc, contentFile, nil
}

// Find a table by name
func (r *Replacer) GetTableElement(doc *etree.Document, name string) *etree.Element {
	tables := doc.FindElements(".//table:table")
	for _, t := range tables {
		if t.SelectAttrValue("table:name", "") == name {
			return t
		}
	}
	return nil
}

// Create a cell with styled paragraphs
func (r *Replacer) GetCell(doc *etree.Document, value, tableStyle, textStyle string) *etree.Element {
	cell := doc.CreateElement("table:table-cell")
	cell.CreateAttr("office:value-type", "string")
	cell.CreateAttr("table:style-name", tableStyle)

	for _, line := range strings.Split(value, "\n") {
		fmt.Println("new node for", line)
		p := doc.CreateElement("text:p")
		p.CreateAttr("text:style-name", textStyle)
		p.SetText(line)
		cell.AddChild(p)
	}
	return cell
}

// Add a new row to the table
func (r *Replacer) TableInsert(doc *etree.Document, tableElement *etree.Element, values []string, designValues []TableEntryStyle) {
	row := doc.CreateElement("table:table-row")
	for i, value := range values {
		cell := r.GetCell(doc, value, designValues[i].CellStyle, designValues[i].TextStyle)
		row.AddChild(cell)
	}
	tableElement.AddChild(row)
}

// Backup last X rows from the table
func (r *Replacer) BackupLastXRows(table *etree.Element, x int) []*etree.Element {
	children := table.ChildElements()
	backup := []*etree.Element{}
	for i := 0; i < x && len(children) > 0; i++ {
		child := children[len(children)-1]
		table.RemoveChild(child)
		backup = append([]*etree.Element{child}, backup...)
		children = table.ChildElements()
	}
	return backup
}

// Reinsert previously backed up rows
func (r *Replacer) ReinsertRows(table *etree.Element, rows []*etree.Element) {
	for _, row := range rows {
		table.AddChild(row)
	}
}

func (r *Replacer) GetStylesOfRow(row *etree.Element) []TableEntryStyle {
	styles := make([]TableEntryStyle, 0, len(row.ChildElements()))
	for _, cell := range row.ChildElements() {
		cellStyle := cell.SelectAttrValue("table:style-name", "")

		// for text style we need to take the first child element
		textStyle := ""
		if len(cell.ChildElements()) > 0 {
			textStyle = cell.ChildElements()[0].SelectAttrValue("text:style-name", "")
		}

		if cellStyle != "" && textStyle != "" {
			styles = append(styles, TableEntryStyle{
				CellStyle: cellStyle,
				TextStyle: textStyle,
			})
		}
	}
	return styles
}

func (r *Replacer) CleanXMLTemplate(xml string) string {
	// translates $<text:span text:style-name="T8">TEXT</text:span> to $TEXT using regex
	re := regexp.MustCompile(`\$<text:span text:style-name="(.*?)">(.*?)</text:span>`)
	xml = re.ReplaceAllString(xml, `\$ $2`)

	// replaces all '\$ ' with $
	xml = strings.ReplaceAll(xml, `\$ `, `$`)

	return xml
}

// Replace values in XML string using a key/value mapping
func (r *Replacer) ReplaceValues(xml string, mapping [][2]string) string {
	xml = r.CleanXMLTemplate(xml)

	for _, pair := range mapping {
		key, edit := pair[0], pair[1]
		if key == "datum" || key == "faellig" {
			parts := strings.Split(edit, "-")
			if len(parts) == 3 {
				edit = fmt.Sprintf("%s.%s.%s", parts[2], parts[1], parts[0])
			}
		}
		fmt.Println("replacing:", "$"+strings.ToUpper(key), edit)
		xml = strings.ReplaceAll(xml, "$"+strings.ToUpper(key), edit)
	}
	fmt.Println(xml)
	return xml
}

// Write new content.xml into a new zip file
func (r *Replacer) WriteContent(srcZipPath, dstZipPath string, xmlContent string) error {
	// Read the original file
	originalZip, err := zip.OpenReader(srcZipPath)
	if err != nil {
		return err
	}
	defer originalZip.Close()

	// Create a backup or a new output file
	newFile, err := os.Create(dstZipPath)
	if err != nil {
		return err
	}
	defer newFile.Close()

	zipWriter := zip.NewWriter(newFile)
	defer zipWriter.Close()

	for _, file := range originalZip.File {
		var writer io.Writer
		var w io.ReadCloser
		if file.Name == "content.xml" {
			writer, err = zipWriter.Create(file.Name)
			if err != nil {
				return err
			}
			_, err = writer.Write([]byte(xmlContent))
			if err != nil {
				return err
			}
		} else {
			writer, err = zipWriter.Create(file.Name)
			if err != nil {
				return err
			}
			w, err = file.Open()
			if err != nil {
				return err
			}
			_, err = io.Copy(writer, w)
			w.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func ConvertODTToPDF(odtPath, pdfPath string) error {
	// executes `libreoffice --convert-to pdf <odtPath> --outdir <pdfPath>`
	cmd := exec.Command("libreoffice", "--convert-to", "pdf", odtPath, "--outdir", filepath.Dir(pdfPath))
	return cmd.Run()
}

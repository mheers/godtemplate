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
	xml = r.NormalizePlaceholderSpans(xml)

	for _, pair := range mapping {
		key, edit := pair[0], pair[1]
		if key == "datum" || key == "faellig" {
			parts := strings.Split(edit, "-")
			if len(parts) == 3 {
				edit = fmt.Sprintf("%s.%s.%s", parts[2], parts[1], parts[0])
			}
		}
		xml = strings.ReplaceAll(xml, "$"+strings.ToUpper(key), edit)
	}
	return xml
}

func (r *Replacer) NormalizePlaceholderSpans(xml string) string {
	data := []byte(xml)
	result := make([]byte, 0, len(data))

	for i := 0; i < len(data); i++ {
		if data[i] != '$' {
			result = append(result, data[i])
			continue
		}

		result = append(result, data[i])
		i++
		for i < len(data) {
			if isPlaceholderChar(data[i]) {
				result = append(result, data[i])
				i++
				continue
			}

			// Check for a span boundary: </text:span>...<text:span ...>
			if data[i] == '<' && hasPrefixAt(data, i, "</text:span") {
				endClose := bytes.IndexByte(data[i:], '>')
				if endClose == -1 {
					result = append(result, data[i:]...)
					return string(result)
				}
				afterClose := i + endClose + 1
				// Skip whitespace between tags
				j := afterClose
				for j < len(data) && (data[j] == ' ' || data[j] == '\t' || data[j] == '\n' || data[j] == '\r') {
					j++
				}
				if j < len(data) && hasPrefixAt(data, j, "<text:span") {
					// Inner span boundary: skip both closing and opening tags
					endOpen := bytes.IndexByte(data[j:], '>')
					if endOpen == -1 {
						result = append(result, data[j:]...)
						return string(result)
					}
					i = j + endOpen + 1
					continue
				}
			}

			// Check for a lone opening span (e.g., $<text:span ...>NAME...)
			if data[i] == '<' && hasPrefixAt(data, i, "<text:span") {
				end := bytes.IndexByte(data[i:], '>')
				if end == -1 {
					result = append(result, data[i:]...)
					return string(result)
				}
				i += end + 1
				continue
			}

			// Any other character (or unpaired </text:span>) ends the placeholder.
			// Back up i so the outer loop's i++ re-processes this character.
			i--
			break
		}
	}

	return string(result)
}

func hasPrefixAt(data []byte, index int, prefix string) bool {
	if index+len(prefix) > len(data) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if data[index+i] != prefix[i] {
			return false
		}
	}
	return true
}

func isPlaceholderChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
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
		if file.Name == "content.xml" {
			// Create a fresh header for the new content.xml;
			// we cannot reuse the original header because the
			// compressed/uncompressed sizes and CRC differ.
			writer, err := zipWriter.CreateHeader(&zip.FileHeader{
				Name:   "content.xml",
				Method: zip.Deflate,
			})
			if err != nil {
				return err
			}
			_, err = writer.Write([]byte(xmlContent))
			if err != nil {
				return err
			}
		} else {
			// Copy the entry verbatim to preserve the exact original
			// ZIP structure (no extra timestamp fields added by Go).
			if err := zipWriter.Copy(file); err != nil {
				return err
			}
		}
	}
	return nil
}

func ConvertODTToPDF(odtPath, pdfPath string) error {
	// executes `libreoffice --headless --convert-to pdf <odtPath> --outdir <dir>`
	outDir := filepath.Dir(pdfPath)
	cmd := exec.Command("libreoffice", "--headless", "--convert-to", "pdf", odtPath, "--outdir", outDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("libreoffice conversion failed: %w\noutput: %s", err, string(output))
	}
	return nil
}

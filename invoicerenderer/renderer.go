package invoicerenderer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/mheers/godtemplate"
)

type Invoice struct {
	Salutation     string
	Name           string
	Street         string
	ZIP            string
	City           string
	DocumentType   string
	DocumentNumber string
	DocumentDate   string
	CustomerNumber string
	Net            float64
	VATRate        float64
	VAT            float64
	Total          float64
	DueDate        string
	TableName      string
}

type InvoiceItem struct {
	Quantity    int
	Unit        string
	Description string
	UnitPrice   float64
	TotalPrice  float64
}

func RenderInvoice(templateInput string, invoice Invoice, items []InvoiceItem, resultOutput string) error {
	r := godtemplate.Replacer{}
	reader, err := r.OpenFile(templateInput)
	if err != nil {
		return fmt.Errorf("failed to open template: %w", err)
	}
	defer reader.Close()

	doc, _, err := r.GetDocument(reader)
	if err != nil {
		return fmt.Errorf("failed to get document: %w", err)
	}

	tableName := invoice.TableName
	last3Rows := r.BackupLastXRows(r.GetTableElement(doc, tableName), 3)
	table := r.GetTableElement(doc, tableName)

	// get the first row of the last3Rows to use as a template for new rows
	if len(last3Rows) == 0 {
		return fmt.Errorf("no rows found in table %s", tableName)
	}
	firstRow := last3Rows[0]
	// -> get the style names from the first row for each column
	styles := r.GetStylesOfRow(firstRow)

	for pos, item := range items {
		values := []string{
			fmt.Sprintf("%d", pos+1),
			fmt.Sprintf("%d", item.Quantity),
			item.Unit,
			item.Description,
			fmt.Sprintf("%.2f €", item.UnitPrice),
			fmt.Sprintf("%.2f €", item.TotalPrice),
		}
		r.TableInsert(doc, table, values, styles)
	}

	r.ReinsertRows(table, last3Rows)

	mapping := [][2]string{
		{"salutation", invoice.Salutation},
		{"name", invoice.Name},
		{"street", invoice.Street},
		{"zip", invoice.ZIP},
		{"city", invoice.City},
		{"documenttype", invoice.DocumentType},
		{"documentnumber", invoice.DocumentNumber},
		{"documentdate", invoice.DocumentDate},
		{"customernumber", invoice.CustomerNumber},
		{"net", fmt.Sprintf("%.2f €", invoice.Net)},
		{"vatrate", fmt.Sprintf("%.2f %%", invoice.VATRate)},
		{"vat", fmt.Sprintf("%.2f €", invoice.VAT)},
		{"total", fmt.Sprintf("%.2f €", invoice.Total)},
		{"duedate", invoice.DueDate},
	}

	xmlContent, err := doc.WriteToString()
	if err != nil {
		return fmt.Errorf("failed to write document to string: %w", err)
	}

	xmlContent = r.ReplaceValues(xmlContent, mapping)

	return r.WriteContent(templateInput, resultOutput, xmlContent)
}

func DecodeBase64JSON(base64JSON string, v interface{}) error {
	// base64 decode the JSON string
	decoded, err := base64.StdEncoding.DecodeString(base64JSON)
	if err != nil {
		return fmt.Errorf("failed to decode base64 JSON: %w", err)
	}
	// unmarshal the JSON into the provided struct
	if err := json.Unmarshal(decoded, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

func ConvertODTToPDF(odtPath, pdfPath string) error {
	return godtemplate.ConvertODTToPDF(odtPath, pdfPath)
}

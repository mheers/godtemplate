package invoicerenderer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mheers/godtemplate"
)

type Invoice struct {
	Salutation     string
	Name           string
	Street         string
	ZIP            string
	City           string
	Telephone      string
	DocumentType   string
	DocumentNumber string
	DocumentDate   string
	DateFormat     string
	CustomerNumber string
	Net            float64
	VATRate        float64
	VAT            float64
	Total          float64
	DueDate        string
	TableName      string
	TableColumns   []string
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
	columns := resolveColumns(invoice.TableColumns, len(styles))
	if columns == nil {
		return fmt.Errorf("unsupported invoice table column count: %d", len(styles))
	}

	for pos, item := range items {
		values, err := buildTableValues(columns, pos, item)
		if err != nil {
			return err
		}
		r.TableInsert(doc, table, values, styles)
	}

	r.ReinsertRows(table, last3Rows)

	documentDate := formatDate(invoice.DocumentDate, invoice.DateFormat)
	dueDate := formatDate(invoice.DueDate, invoice.DateFormat)

	mapping := [][2]string{
		{"salutation", invoice.Salutation},
		{"name", invoice.Name},
		{"street", invoice.Street},
		{"zip", invoice.ZIP},
		{"city", invoice.City},
		{"tel", invoice.Telephone},
		{"documenttype", invoice.DocumentType},
		{"documentnumber", invoice.DocumentNumber},
		{"documentdate", documentDate},
		{"customernumber", invoice.CustomerNumber},
		{"net", fmt.Sprintf("%.2f €", invoice.Net)},
		{"vatrate", fmt.Sprintf("%.2f %%", invoice.VATRate)},
		{"vat", fmt.Sprintf("%.2f €", invoice.VAT)},
		{"total", fmt.Sprintf("%.2f €", invoice.Total)},
		{"duedate", dueDate},
	}

	xmlContent, err := doc.WriteToString()
	if err != nil {
		return fmt.Errorf("failed to write document to string: %w", err)
	}

	xmlContent = r.ReplaceValues(xmlContent, mapping)

	return r.WriteContent(templateInput, resultOutput, xmlContent)
}

func resolveColumns(configured []string, styleCount int) []string {
	if len(configured) > 0 {
		return configured
	}

	switch styleCount {
	case 4:
		return []string{"menge", "text", "betrag", "gesamt"}
	case 5:
		return []string{"position", "menge", "text", "betrag", "gesamt"}
	case 6:
		return []string{"position", "menge", "einheit", "text", "betrag", "gesamt"}
	default:
		return nil
	}
}

func buildTableValues(columns []string, position int, item InvoiceItem) ([]string, error) {
	values := make([]string, 0, len(columns))
	for _, column := range columns {
		switch column {
		case "position", "pos", "nr":
			values = append(values, fmt.Sprintf("%d", position+1))
		case "menge", "qty", "quantity":
			values = append(values, fmt.Sprintf("%d", item.Quantity))
		case "einheit", "unit":
			values = append(values, item.Unit)
		case "text", "beschreibung", "description":
			values = append(values, item.Description)
		case "betrag", "unitprice", "price":
			values = append(values, fmt.Sprintf("%.2f €", item.UnitPrice))
		case "gesamt", "total", "totalprice":
			values = append(values, fmt.Sprintf("%.2f €", item.TotalPrice))
		default:
			return nil, fmt.Errorf("unsupported invoice table column: %s", column)
		}
	}

	return values, nil
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

func formatDate(value, layout string) string {
	if value == "" || layout == "" {
		return value
	}

	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return value
	}

	return parsed.Format(layout)
}

# gODTemplate

> A Go library for manipulating OpenDocumentText files

## Features

- Render invoices from ODT templates
- Supports JSON and base64 encoded data
- Provides a CLI tool and a server implementation
- Can be used as a library in Go applications
- Docker image available for easy deployment

## Usage

### As Library

```go
package main

import (
	"github.com/mheers/godtemplate/invoicerenderer"
)

func main() {
	err := invoicerenderer.RenderInvoice(
		"templates/template.odt",
		invoicerenderer.Invoice{
			Salutation:     "Mr.",
			Name:           "John Doe",
			Street:         "123 Main St",
			ZIP:            "12345",
			City:           "Anytown",
			DocumentType:   "Invoice",
			DocumentNumber: "1000251",
			DocumentDate:   "2025-05-01",
			CustomerNumber: "C123456",
			Net:            36.00,
			VATRate:        12.0,
			VAT:            4.32,
			Total:          40.32,
			DueDate:        "2025-06-01",
			TableName:      "Listing",
		},
		[]invoicerenderer.InvoiceItem{
			{Description: "Water", Quantity: 2, Unit: "L", UnitPrice: 15.00, TotalPrice: 30.00},
			{Description: "Shoes", Quantity: 3, Unit: "pcs", UnitPrice: 2.00, TotalPrice: 6.00},
		},
		"/tmp/output_invoice.odt",
	)

	if err != nil {
		panic(err)
	}
}
```

### As CLI Tool

```bash
godtemplate render --template templates/template.odt --output /tmp/output_invoice.odt --invoice $(cat example/data.json| jq .invoice -r | base64 -w 0) --items $(cat example/data.json | jq .items -r | base64 -w 0)
```

### As Docker Image

```bash
docker run --rm -v $(pwd):/data mheers/godtemplate render --template /data/templates/template.odt --output /data/output_invoice.odt --invoice <base64-encoded-invoice-json> --items <base64-encoded-items-json>
```

Instead of odt, you can also specify `pdf` as output format. The tool will convert the ODT file to PDF using LibreOffice. This requires LibreOffice to be installed on your system. LibreOffice is already included in the Docker image.

### As Server

You can also run the tool as a server that provides HTTP endpoints to render invoices as PDF documents from ODT templates.

For detailed instructions on how to run the server, see the [SERVER.md](SERVER.md) file.

## License
This project is licensed under the MIT License.

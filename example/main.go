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

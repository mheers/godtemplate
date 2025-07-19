package main

import (
	"fmt"

	"github.com/mheers/godtemplate/invoicerenderer"
	"github.com/spf13/cobra"
)

var (
	templateFile   string
	outputFile     string
	invoiceB64Json string
	itemsB64Json   string

	renderCmd = &cobra.Command{
		Use:     "render",
		Short:   "Render an invoice template",
		Long:    `Render an invoice template with provided data and items.`,
		Example: `godtemplate render --template templates/template.odt --output /tmp/output_invoice.[odt|pdf] --invoice <base64-encoded-invoice-json> --items <base64-encoded-items-json>`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return render()
		},
	}
)

func init() {
	renderCmd.Flags().StringVarP(&templateFile, "template", "t", "templates/template.odt", "Input file (must be specified)")
	renderCmd.Flags().StringVarP(&outputFile, "output", "o", "output.pdf", "Output file (pdf or odt) (default: output.pdf)")
	renderCmd.Flags().StringVarP(&invoiceB64Json, "invoice", "i", "", "Invoice data in Bas64 JSON format")
	renderCmd.Flags().StringVarP(&itemsB64Json, "items", "l", "", "List of invoice items in Bas64 JSON format")
	renderCmd.MarkFlagRequired("template")
	renderCmd.MarkFlagRequired("output")
	renderCmd.MarkFlagRequired("invoice")
	renderCmd.MarkFlagRequired("items")
}

func render() error {
	if templateFile == "" {
		return fmt.Errorf("template file must be specified")
	}

	if outputFile == "" {
		return fmt.Errorf("output file must be specified")
	}

	// check if the output file has a valid extension
	if outputFile[len(outputFile)-4:] != ".odt" && outputFile[len(outputFile)-4:] != ".pdf" {
		return fmt.Errorf("output file must have a .odt or .pdf extension")
	}

	isPDF := outputFile[len(outputFile)-4:] == ".pdf"
	if isPDF {
		outputFile = outputFile[:len(outputFile)-4] + ".odt" // convert to odt for rendering
	}

	var invoiceData invoicerenderer.Invoice
	if err := invoicerenderer.DecodeBase64JSON(invoiceB64Json, &invoiceData); err != nil {
		return fmt.Errorf("failed to decode invoice JSON: %w", err)
	}

	var itemsData []invoicerenderer.InvoiceItem
	if err := invoicerenderer.DecodeBase64JSON(itemsB64Json, &itemsData); err != nil {
		return fmt.Errorf("failed to decode items JSON: %w", err)
	}

	if err := invoicerenderer.RenderInvoice(
		templateFile,
		invoiceData,
		itemsData,
		outputFile,
	); err != nil {
		return fmt.Errorf("failed to render invoice: %w", err)
	}

	if isPDF {
		// Convert the odt file to pdf
		if err := invoicerenderer.ConvertODTToPDF(outputFile, outputFile[:len(outputFile)-4]+".pdf"); err != nil {
			return fmt.Errorf("failed to convert odt to pdf: %w", err)
		}
		fmt.Println("Invoice rendered and converted to PDF:", outputFile[:len(outputFile)-4]+".pdf")
	} else {
		fmt.Println("Invoice rendered:", outputFile)
	}
	return nil
}

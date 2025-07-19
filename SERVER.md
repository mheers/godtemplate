# Godtemplate Server

This server implementation provides HTTP endpoints to render invoices as PDF documents from ODT templates.

## Usage

Start the server:
```bash
./godtemplate server --port 8080 --template templates/template.odt
```

## Endpoints

### POST /render
Renders an invoice from JSON data and returns a PDF.

**Request Body:**
```json
{
    "invoice": {
        "Salutation": "Mr.",
        "Name": "John Doe",
        "Street": "123 Main St",
        "ZIP": "12345",
        "City": "Anytown",
        "DocumentType": "Invoice",
        "DocumentNumber": "INV-001",
        "DocumentDate": "2025-07-19",
        "CustomerNumber": "CUST-001",
        "Net": 100.00,
        "VATRate": 19.00,
        "VAT": 19.00,
        "Total": 119.00,
        "DueDate": "2025-08-19",
        "TableName": "Listing"
    },
    "items": [
        {
            "Description": "Web Development Services",
            "Quantity": 1,
            "Unit": "hrs",
            "UnitPrice": 100.00,
            "TotalPrice": 100.00
        }
    ]
}
```

**Response:** PDF file download

### POST /render-base64
Renders an invoice from base64-encoded JSON data and returns a PDF.

**Request Body:** Base64 encoded JSON string

### GET /health
Health check endpoint.

**Response:**
```json
{
    "status": "healthy",
    "timestamp": "2025-07-19T10:30:00Z",
    "template": "templates/template.odt"
}
```

## Example

```bash
# Test with curl
curl -X POST \
  -H "Content-Type: application/json" \
  -d @examples/data.json \
  http://localhost:8080/render \
  --output invoice.pdf
```

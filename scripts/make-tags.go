// scripts/make-tags.go
//go:build ignore

// make-tags generates a printable A4 sheet of provenance QR tags.
//
// Usage:
//   go run scripts/make-tags.go -codes CODE1,CODE2,CODE3 -base-url https://kalakriti.example.com -output tags.html
package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"strings"
)

var tmpl = template.Must(template.New("tags").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Provenance Tags</title>
    <style>
        @page {
            size: A4;
            margin: 10mm;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            margin: 0;
            padding: 0;
        }
        .sheet {
            width: 190mm;
            height: 277mm;
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            grid-template-rows: repeat(4, 1fr);
            gap: 5mm;
            padding: 5mm;
        }
        .tag {
            border: 1px solid #ccc;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            padding: 5mm;
            page-break-inside: avoid;
        }
        .tag img {
            width: 40mm;
            height: 40mm;
            margin-bottom: 3mm;
        }
        .code {
            font-size: 14pt;
            font-weight: bold;
            letter-spacing: 0.5pt;
            margin-bottom: 2mm;
        }
        .url {
            font-size: 8pt;
            color: #666;
            word-break: break-all;
        }
        @media print {
            .sheet {
                page-break-after: always;
            }
        }
    </style>
</head>
<body>
    <div class="sheet">
        {{range .Tags}}
        <div class="tag">
            <img src="https://api.qrserver.com/v1/create-qr-code/?size=400x400&data={{.URL}}" alt="QR Code">
            <div class="code">{{.Code}}</div>
            <div class="url">{{.URL}}</div>
        </div>
        {{end}}
    </div>
</body>
</html>
`))

type tag struct {
	Code string
	URL  string
}

func main() {
	var (
		codes   = flag.String("codes", "", "Comma-separated short codes")
		baseURL = flag.String("base-url", "https://kalakriti.example.com", "Base verification URL")
		output  = flag.String("output", "tags.html", "Output HTML file")
	)
	flag.Parse()

	if *codes == "" {
		fmt.Fprintln(os.Stderr, "Usage: go run scripts/make-tags.go -codes CODE1,CODE2,CODE3")
		os.Exit(1)
	}

	codeList := strings.Split(*codes, ",")
	tags := make([]tag, len(codeList))
	for i, code := range codeList {
		code = strings.TrimSpace(code)
		tags[i] = tag{
			Code: code,
			URL:  fmt.Sprintf("%s/v/%s", *baseURL, code),
		}
	}

	f, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if err := tmpl.Execute(f, map[string]any{"Tags": tags}); err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering template: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s with %d tags\n", *output, len(tags))
}

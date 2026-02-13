# Data Ingestion Schemas

JSON Schema definitions for validating ingestion API requests.

## Structure

```
schemas/
  ycharts/
    key_stats.json          - YCharts Key Stats page metrics
    financials.json         - YCharts Financials page (income, balance, cash flow)
    valuation.json          - YCharts Valuation page
    performance.json        - YCharts Performance page
  seekingalpha/
    ratings.json            - SeekingAlpha ratings and grades
    analysis.json           - SeekingAlpha analysis articles
```

## Schema Format

All schemas follow JSON Schema Draft 07 specification.

### Required Fields

Every schema must include:
- `$schema`: JSON Schema version
- `$id`: Unique schema identifier
- `title`: Human-readable schema name
- `description`: Schema purpose
- `type`: "object"
- `required`: Array of required field names
- `properties`: Field definitions

### Field Types

**Integers** (dollar amounts):
- Revenue, assets, liabilities, etc.
- Stored as absolute values (e.g., 187140000000 for $187.14B)

**Decimals** (ratios, percentages):
- Margins, growth rates, returns, etc.
- Stored as decimal values (e.g., 0.6249 for 62.49%)

**Strings**:
- Dates: ISO 8601 format (YYYY-MM-DD)
- Timestamps: ISO 8601 with timezone (YYYY-MM-DDTHH:MM:SSZ)

**Nullable Fields**:
- All metrics should be nullable: `"type": ["number", "null"]`
- Missing data = `null`, not `0`

## Usage in Go

```go
import (
    "encoding/json"
    "github.com/xeipuuv/gojsonschema"
)

// Load schema
schemaPath := "schemas/ycharts/key_stats.json"
schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)

// Validate request
documentLoader := gojsonschema.NewGoLoader(requestData)
result, err := gojsonschema.Validate(schemaLoader, documentLoader)

if err != nil {
    return fmt.Errorf("schema validation error: %w", err)
}

if !result.Valid() {
    for _, desc := range result.Errors() {
        fmt.Printf("- %s\n", desc)
    }
    return fmt.Errorf("request validation failed")
}
```

## Adding New Schemas

1. Create schema file in appropriate subdirectory
2. Follow the field type conventions above
3. Add schema path to API handler registration
4. Update this README

## Validation Rules

### Required vs Optional

- **Always required**: `collected_at`, `source_url`
- **Optional**: All metric fields (may be null if not available on source page)

### Data Types

- Use `integer` for whole numbers (counts, dollar amounts)
- Use `number` for decimals (ratios, percentages, scores)
- Use `string` with `format: "date"` for dates
- Use `string` with `format: "date-time"` for timestamps

### Constraints

- Add `maxLength` for string fields
- Add `minimum`/`maximum` for bounded numeric fields
- Add `pattern` for formatted strings (e.g., ticker symbols)

## Testing Schemas

Test schemas before deploying:

```bash
# Validate schema itself
jsonschema -i schemas/ycharts/key_stats.json

# Validate example data
jsonschema -i example_data.json schemas/ycharts/key_stats.json
```

## References

- JSON Schema Specification: https://json-schema.org/
- Go Validator Library: https://github.com/xeipuuv/gojsonschema

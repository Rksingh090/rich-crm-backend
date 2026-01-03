package module

import (
	"go-crm/internal/common/models"
)

// Aliases to keep backward compatibility during refactor
type Module = models.Entity
type ModuleField = models.ModuleField
type FieldType = models.FieldType
type SelectOptions = models.SelectOptions
type LookupDef = models.LookupDef

const (
	FieldTypeText        = models.FieldTypeText
	FieldTypeNumber      = models.FieldTypeNumber
	FieldTypeDate        = models.FieldTypeDate
	FieldTypeBoolean     = models.FieldTypeBoolean
	FieldTypeLookup      = models.FieldTypeLookup
	FieldTypeEmail       = models.FieldTypeEmail
	FieldTypePhone       = models.FieldTypePhone
	FieldTypeFile        = models.FieldTypeFile
	FieldTypeURL         = models.FieldTypeURL
	FieldTypeTextArea    = models.FieldTypeTextArea
	FieldTypeSelect      = models.FieldTypeSelect
	FieldTypeMultiSelect = models.FieldTypeMultiSelect
	FieldTypeCurrency    = models.FieldTypeCurrency
	FieldTypeImage       = models.FieldTypeImage
)

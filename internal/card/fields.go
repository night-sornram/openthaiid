package card

// Field identifies an approved, normalized card field.
type Field string

const (
	FieldCitizenID Field = "citizenId"
	FieldThaiName  Field = "thaiName"
	FieldBirthDate Field = "birthDate"
)

// MVPFields are the only fields implemented in the first release.
var MVPFields = []Field{FieldCitizenID, FieldThaiName, FieldBirthDate}

// DefaultFields are requested by the SDK when the application does not override.
var DefaultFields = []Field{FieldCitizenID, FieldThaiName, FieldBirthDate}

// Supported reports whether f is in the MVP allowlist.
func Supported(f Field) bool {
	switch f {
	case FieldCitizenID, FieldThaiName, FieldBirthDate:
		return true
	default:
		return false
	}
}

// NormalizeFields validates and de-duplicates a requested field list.
// Unknown fields are rejected. An empty list is rejected; callers must
// provide an explicit set.
func NormalizeFields(fields []Field) ([]Field, error) {
	if len(fields) == 0 {
		return nil, NewError(ErrInvalidRequest, "fields must be non-empty")
	}
	seen := make(map[Field]struct{}, len(fields))
	out := make([]Field, 0, len(fields))
	for _, f := range fields {
		if !Supported(f) {
			return nil, NewError(ErrUnsupportedField, "unsupported field")
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out, nil
}

// Subset reports whether want is a subset of allowed.
func Subset(want, allowed []Field) bool {
	set := make(map[Field]struct{}, len(allowed))
	for _, f := range allowed {
		set[f] = struct{}{}
	}
	for _, f := range want {
		if _, ok := set[f]; !ok {
			return false
		}
	}
	return true
}

// CardDate is a partial Gregorian date.
// Unknown components stay null; they are never defaulted to 1.
// Year is always Common Era after parsing.
type CardDate struct {
	Year  *int `json:"year"`
	Month *int `json:"month"`
	Day   *int `json:"day"`
}

// ThaiName is a structured Thai personal name.
type ThaiName struct {
	Title      *string `json:"title"`
	GivenName  *string `json:"givenName"`
	MiddleName *string `json:"middleName"`
	FamilyName *string `json:"familyName"`
	FullName   string  `json:"fullName"`
}

// FieldValues holds normalized field results for one read.
// Unrequested fields are omitted from JSON. Requested but unavailable
// fields marshal as null and carry a warning.
type FieldValues struct {
	CitizenID *string   `json:"citizenId,omitempty"`
	ThaiName  *ThaiName `json:"thaiName,omitempty"`
	BirthDate *CardDate `json:"birthDate,omitempty"`

	requested []Field
}

// NewFieldValues records which fields were requested so unavailable ones
// can be marshaled as explicit nulls.
func NewFieldValues(requested []Field) *FieldValues {
	return &FieldValues{requested: append([]Field(nil), requested...)}
}

// MarshalJSON emits only requested fields. Present values are encoded;
// missing requested values are null.
func (f FieldValues) MarshalJSON() ([]byte, error) {
	if f.requested == nil {
		// Fallback for values built without NewFieldValues (tests, copies).
		type alias FieldValues
		return jsonMarshal((*alias)(&f))
	}
	m := make(map[string]any, 3)
	for _, field := range f.requested {
		switch field {
		case FieldCitizenID:
			m["citizenId"] = f.CitizenID
		case FieldThaiName:
			m["thaiName"] = f.ThaiName
		case FieldBirthDate:
			m["birthDate"] = f.BirthDate
		}
	}
	return jsonMarshal(m)
}

// Warning is a non-fatal field-level problem.
type Warning struct {
	Field   Field  `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ReadResult is the successful outcome of a selected-field read.
type ReadResult struct {
	ReaderID      string      `json:"readerId"`
	CardSessionID string      `json:"cardSessionId"`
	Fields        FieldValues `json:"fields"`
	Warnings      []Warning   `json:"warnings"`
}

package card

// Field identifies an approved, normalized card field.
type Field string

const (
	FieldCitizenID   Field = "citizenId"
	FieldThaiName    Field = "thaiName"
	FieldEnglishName Field = "englishName"
	FieldBirthDate   Field = "birthDate"
	FieldGender      Field = "gender"
	FieldIssuer      Field = "issuer"
	FieldIssueDate   Field = "issueDate"
	FieldExpireDate  Field = "expireDate"
	FieldAddress     Field = "address"
	FieldPortrait    Field = "portrait"
)

// GenderValue is the normalized gender after mapping the card's 1-byte code.
const (
	GenderMale   = "male"
	GenderFemale = "female"
)

// DefaultFields are requested by the SDK when the application does not override.
// This remains the original three-field set (least privilege).
var DefaultFields = []Field{FieldCitizenID, FieldThaiName, FieldBirthDate}

// MVPFields is the original three-field set. Prefer DefaultFields.
var MVPFields = []Field{FieldCitizenID, FieldThaiName, FieldBirthDate}

// TextualFields is every textual field this profile can return.
// Order follows the cited APDU table (dotnetthailand APDU.md).
var TextualFields = []Field{
	FieldCitizenID,
	FieldThaiName,
	FieldEnglishName,
	FieldBirthDate,
	FieldGender,
	FieldIssuer,
	FieldIssueDate,
	FieldExpireDate,
	FieldAddress,
}

// SupportedFields is every field this profile can return, including the
// optional portrait. Default grant stays DefaultFields (no photo).
var SupportedFields = []Field{
	FieldCitizenID,
	FieldThaiName,
	FieldEnglishName,
	FieldBirthDate,
	FieldGender,
	FieldIssuer,
	FieldIssueDate,
	FieldExpireDate,
	FieldAddress,
	FieldPortrait,
}

// Supported reports whether f is in the profile allowlist.
func Supported(f Field) bool {
	switch f {
	case FieldCitizenID, FieldThaiName, FieldEnglishName, FieldBirthDate,
		FieldGender, FieldIssuer, FieldIssueDate, FieldExpireDate, FieldAddress,
		FieldPortrait:
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

// ThaiName is a structured personal name (Thai or English card block).
type ThaiName struct {
	Title      *string `json:"title"`
	GivenName  *string `json:"givenName"`
	MiddleName *string `json:"middleName"`
	FamilyName *string `json:"familyName"`
	FullName   string  `json:"fullName"`
}

// Gender is a 1-byte card code plus a normalized value.
// Community convention (not stated in APDU.md): "1" = male, "2" = female.
// Unrecognized codes keep Code and set Value to null with a warning.
type Gender struct {
	Code  string  `json:"code"`
	Value *string `json:"value"`
}

// ThaiAddress is a structured registered address.
// Community convention: one 100-byte '#'-delimited block:
//
//	houseNo # villageNo # alley # street # subdistrict # district # province
//
// Component order is a hypothesis pending physical-card confirmation.
type ThaiAddress struct {
	HouseNo     *string `json:"houseNo"`
	VillageNo   *string `json:"villageNo"`
	Alley       *string `json:"alley"`
	Street      *string `json:"street"`
	Subdistrict *string `json:"subdistrict"`
	District    *string `json:"district"`
	Province    *string `json:"province"`
	FullAddress string  `json:"fullAddress"`
}

// FieldValues holds normalized field results for one read.
// Unrequested fields are omitted from JSON. Requested but unavailable
// fields marshal as null and carry a warning.
type FieldValues struct {
	CitizenID   *string      `json:"citizenId,omitempty"`
	ThaiName    *ThaiName    `json:"thaiName,omitempty"`
	EnglishName *ThaiName    `json:"englishName,omitempty"`
	BirthDate   *CardDate    `json:"birthDate,omitempty"`
	Gender      *Gender      `json:"gender,omitempty"`
	Issuer      *string      `json:"issuer,omitempty"`
	IssueDate   *CardDate    `json:"issueDate,omitempty"`
	ExpireDate  *CardDate    `json:"expireDate,omitempty"`
	Address     *ThaiAddress `json:"address,omitempty"`
	Portrait    *Portrait    `json:"portrait,omitempty"`

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
	m := make(map[string]any, len(f.requested))
	for _, field := range f.requested {
		switch field {
		case FieldCitizenID:
			m["citizenId"] = f.CitizenID
		case FieldThaiName:
			m["thaiName"] = f.ThaiName
		case FieldEnglishName:
			m["englishName"] = f.EnglishName
		case FieldBirthDate:
			m["birthDate"] = f.BirthDate
		case FieldGender:
			m["gender"] = f.Gender
		case FieldIssuer:
			m["issuer"] = f.Issuer
		case FieldIssueDate:
			m["issueDate"] = f.IssueDate
		case FieldExpireDate:
			m["expireDate"] = f.ExpireDate
		case FieldAddress:
			m["address"] = f.Address
		case FieldPortrait:
			m["portrait"] = f.Portrait
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

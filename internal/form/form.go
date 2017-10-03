package form

type Form struct {
	fields       []Field
	focusedField int
}

func (f *Form) Values() map[string]string {
	result := make(map[string]string)
	for _, field := range f.fields {
		result[field.Code] = field.Value
	}
	return result
}

func (f *Form) Next() {
	idx := f.focusedField + 1
	num := len(f.fields)
	if num == 0 {
		return
	}
	if idx >= num {
		idx = 0
	}
	f.focusedField = idx
}

func (f *Form) Previous() {
	idx := f.focusedField - 1
	num := len(f.fields)
	if num == 0 {
		return
	}
	if idx < 0 {
		idx = num - 1
	}
	f.focusedField = idx
}

func (f *Form) IsFocused(code string) bool {
	return f.FocusedField() == code
}

func (f *Form) FocusedField() string {
	if f.focusedField == -1 || f.focusedField > len(f.fields)-1 {
		return ""
	}
	return f.fields[f.focusedField].Code
}

func (f *Form) IsValid() bool {
	for _, fld := range f.fields {
		if !fld.IsValid() {
			return false
		}
	}
	return true
}

func (f *Form) Fields() []Field {
	res := make([]Field, 0, len(f.fields))
	for _, fld := range f.fields {
		res = append(res, fld)
	}
	return res
}

type Field struct {
	Code       string
	Label      string
	Value      string
	IsRequired bool
	Error      string
}

func (f *Field) IsValid() bool {
	return f.Error == ""
}

func NewForm(fields []Field) *Form {
	f := Form{
		fields: make([]Field, 0, len(fields)),
	}
	for _, fld := range fields {
		f.fields = append(f.fields, fld)
	}
	return &f
}

func (f *Form) Validate() bool {
	var invalid bool
	for idx := range f.fields {
		if f.fields[idx].IsRequired && f.fields[idx].Value == "" {
			f.fields[idx].Error = "This field is required."
			invalid = true
		} else {
			f.fields[idx].Error = ""
		}
	}
	return !invalid
}

func (f *Form) Value(field string) string {
	for _, fld := range f.fields {
		if fld.Code == field {
			return fld.Value
		}
	}
	return ""
}

func (f *Form) SetValue(field string, value string) {
	for idx, fld := range f.fields {
		if fld.Code == field {
			f.fields[idx].Value = value
			// fld.Value = value
		}
	}
}

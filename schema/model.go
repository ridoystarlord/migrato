package schema

type Model struct {
	TableName string
	Columns   []Column
}

type Column struct {
	Name     string
	Type     string
	Primary  bool
	Unique   bool
	Default  *string
}

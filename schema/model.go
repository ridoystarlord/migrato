package schema

type Model struct {
	TableName string
	Columns   []Column
	Relations []Relation
	Indexes   []Index
}

type Column struct {
	Name     string
	Type     string
	Primary  bool
	Unique   bool
	NotNull  bool
	Default  *string
	ForeignKey *ForeignKey
	Index    *IndexConfig
}

type IndexConfig struct {
	Name    string
	Columns []string
	Unique  bool
	Type    string // btree, hash, gin, etc.
}

type ForeignKey struct {
	ReferencesTable  string
	ReferencesColumn string
	OnDelete         string // CASCADE, SET NULL, RESTRICT, etc.
	OnUpdate         string // CASCADE, SET NULL, RESTRICT, etc.
}

type Relation struct {
	Name           string
	Type           RelationType
	FromTable      string
	FromColumn     string
	ToTable        string
	ToColumn       string
	JunctionTable  string // for many-to-many relationships
}

type RelationType string

const (
	OneToOne   RelationType = "one-to-one"
	OneToMany  RelationType = "one-to-many"
	ManyToOne  RelationType = "many-to-one"
	ManyToMany RelationType = "many-to-many"
)

type Index struct {
	Name    string
	Table   string
	Columns []string
	Unique  bool
	Type    string // btree, hash, gin, etc.
}

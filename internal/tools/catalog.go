package tools

// SideEffectClass classifies a tool's blast radius (docs/04 §3). It governs
// whether confirmation/approval is required before invocation.
type SideEffectClass string

const (
	ReadOnly             SideEffectClass = "read_only"
	ReversibleWrite      SideEffectClass = "reversible_write"
	IrreversibleMonetary SideEffectClass = "irreversible_or_monetary"
)

// Tool is a catalog entry. The catalog is descriptive metadata; actual
// execution lives in the exec package, and permission is enforced via authz.
type Tool struct {
	Key             string          `json:"key"`
	DisplayName     string          `json:"display_name"`
	SideEffectClass SideEffectClass `json:"side_effect_class"`
	Description     string          `json:"description"`
	// ScopeDimensions documents which scope keys bound this tool (e.g. amount_cap).
	ScopeDimensions []string `json:"scope_dimensions,omitempty"`
}

// Catalog is an in-memory tool registry.
type Catalog struct {
	tools map[string]Tool
}

func NewCatalog() *Catalog {
	return &Catalog{tools: map[string]Tool{}}
}

func (c *Catalog) Register(t Tool) { c.tools[t.Key] = t }

func (c *Catalog) Get(key string) (Tool, bool) {
	t, ok := c.tools[key]
	return t, ok
}

func (c *Catalog) All() []Tool {
	out := make([]Tool, 0, len(c.tools))
	for _, t := range c.tools {
		out = append(out, t)
	}
	return out
}

// DefaultCatalog returns the seeded tool set for the reference workflow.
func DefaultCatalog() *Catalog {
	c := NewCatalog()
	c.Register(Tool{
		Key: "fare.lookup", DisplayName: "運賃照会", SideEffectClass: ReadOnly,
		Description: "経路から運賃を算出する（参照のみ）",
	})
	c.Register(Tool{
		Key: "pre_application.submit", DisplayName: "事前申請の提出", SideEffectClass: IrreversibleMonetary,
		Description: "事前申請を投稿する（不可逆）",
	})
	c.Register(Tool{
		Key: "payment.execute", DisplayName: "精算（支払）実行", SideEffectClass: IrreversibleMonetary,
		Description: "精算（送金）を実行する（不可逆・冪等必須）", ScopeDimensions: []string{"amount_cap"},
	})
	return c
}

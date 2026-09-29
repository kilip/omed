package schema

import "entgo.io/ent"

// Workspace holds the schema definition for the Workspace entity.
type Workspace struct {
	ent.Schema
}

func (Workspace) Mixin() []ent.Mixin {
	return []ent.Mixin{
		IDV7Mixin{},
	}
}

// Fields of the Workspace.
func (Workspace) Fields() []ent.Field {
	return nil
}

// Edges of the Workspace.
func (Workspace) Edges() []ent.Edge {
	return nil
}

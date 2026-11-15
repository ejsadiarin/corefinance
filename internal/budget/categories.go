package budget

import (
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ejsadiarin/corefinance/internal/category"
)

// categoryTarget is where an imported record's category lands (ADR-0002):
// the system category of the same name, or `other` plus a tag of the old name.
type categoryTarget struct {
	ID  uuid.UUID
	Tag string
}

type namedCategory struct {
	ID   uuid.UUID
	Name string
}

// categoryMapper maps an export file's category ids onto one kind's system
// list.
type categoryMapper struct {
	list  []category.System
	other uuid.UUID
	byID  map[uuid.UUID]categoryTarget
}

func newCategoryMapper(list []category.System, other uuid.UUID, cats []namedCategory) categoryMapper {
	m := categoryMapper{list: list, other: other, byID: make(map[uuid.UUID]categoryTarget, len(cats))}
	for _, c := range cats {
		if sys, ok := category.ByName(list, c.Name); ok {
			m.byID[c.ID] = categoryTarget{ID: sys.ID}
			continue
		}
		t := categoryTarget{ID: other}
		t.Tag = strings.TrimSpace(c.Name)
		m.byID[c.ID] = t
	}
	return m
}

// target resolves a record's category id from the file. An id the file lists
// maps by its name; a system id of this kind is kept; anything else is no
// category, as it showed before.
func (m categoryMapper) target(id pgtype.UUID) categoryTarget {
	if !id.Valid {
		return categoryTarget{}
	}
	if t, ok := m.byID[id.Bytes]; ok {
		return t
	}
	for _, c := range m.list {
		if c.ID == uuid.UUID(id.Bytes) {
			return categoryTarget{ID: c.ID}
		}
	}
	return categoryTarget{}
}

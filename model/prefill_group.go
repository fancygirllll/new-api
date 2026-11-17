package model

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// PrefillGroup stores reusable "group" information, such as model groups, tag groups, and endpoint groups.
// Name is unique and displayed in frontend dropdowns.
// Type distinguishes the group category; possible values: model, tag, endpoint.
// Items holds the string collection for the type as a JSON array, for example:
// ["gpt-4o", "gpt-3.5-turbo"]
// The design follows 3NF, avoids redundancy, and allows flexible extension.

// JSONValue is based on json.RawMessage and supports reading []byte and string from the database
type JSONValue json.RawMessage

// Value implements driver.Valuer for database writes
func (j JSONValue) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return []byte(j), nil
}

// Scan implements sql.Scanner, compatible with types returned by different drivers
func (j *JSONValue) Scan(value interface{}) error {
	switch v := value.(type) {
	case nil:
		*j = nil
		return nil
	case []byte:
		// Copy the underlying bytes to avoid retaining the source buffer
		b := make([]byte, len(v))
		copy(b, v)
		*j = JSONValue(b)
		return nil
	case string:
		*j = JSONValue([]byte(v))
		return nil
	default:
		// Try serializing other types as JSON
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		*j = JSONValue(b)
		return nil
	}
}

// MarshalJSON ensures encoding behaves like json.RawMessage
func (j JSONValue) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON ensures decoding behaves like json.RawMessage
func (j *JSONValue) UnmarshalJSON(data []byte) error {
	if data == nil {
		*j = nil
		return nil
	}
	b := make([]byte, len(data))
	copy(b, data)
	*j = JSONValue(b)
	return nil
}

type PrefillGroup struct {
	Id          int            `json:"id"`
	Name        string         `json:"name" gorm:"size:64;not null;uniqueIndex:uk_prefill_name,where:deleted_at IS NULL"`
	Type        string         `json:"type" gorm:"size:32;index;not null"`
	Items       JSONValue      `json:"items" gorm:"type:json"`
	Description string         `json:"description,omitempty" gorm:"type:varchar(255)"`
	CreatedTime int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// Insert creates a new group
func (g *PrefillGroup) Insert() error {
	now := common.GetTimestamp()
	g.CreatedTime = now
	g.UpdatedTime = now
	return DB.Create(g).Error
}

// IsPrefillGroupNameDuplicated checks for duplicate group names (excluding its own ID)
func IsPrefillGroupNameDuplicated(id int, name string) (bool, error) {
	if name == "" {
		return false, nil
	}
	var cnt int64
	err := DB.Model(&PrefillGroup{}).Where("name = ? AND id <> ?", name, id).Count(&cnt).Error
	return cnt > 0, err
}

// Update updates the group
func (g *PrefillGroup) Update() error {
	g.UpdatedTime = common.GetTimestamp()
	return DB.Save(g).Error
}

// DeleteByID deletes a group by ID
func DeletePrefillGroupByID(id int) error {
	return DB.Delete(&PrefillGroup{}, id).Error
}

// GetAllPrefillGroups returns all groups, optionally filtered by type (empty returns all)
func GetAllPrefillGroups(groupType string) ([]*PrefillGroup, error) {
	var groups []*PrefillGroup
	query := DB.Model(&PrefillGroup{})
	if groupType != "" {
		query = query.Where("type = ?", groupType)
	}
	if err := query.Order("updated_time DESC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

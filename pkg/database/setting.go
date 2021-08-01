package database

type setting struct {
	Name  string `gorm:"primaryKey"`
	Value string
}

func (d Database) GetSetting(name string) string {
	s := setting{}
	tx := d.db.Where(&setting{Name: name}).First(&s)
	if tx.Error != nil {
		return ""
	}
	return s.Value
}

func (d Database) SaveSetting(name, value string) string {
	s := setting{Name: name, Value: value}
	tx := d.db.Where(&setting{Name: name}).FirstOrCreate(&s)
	if tx.Error != nil {
		return ""
	}
	s.Value = value
	tx = d.db.Save(s)
	if tx.Error != nil {
		return ""
	}
	return s.Value
}

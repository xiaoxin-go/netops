package model

import "time"

type TInnerPolicy struct {
	Id              int       `gorm:"primary_key" json:"id"`
	Region          string    `gorm:"column:region" json:"region" binding:"-"`
	SourceName      string    `gorm:"column:source_name" json:"source_name" binding:"-"`
	DestinationName string    `gorm:"column:destination_name" json:"destination_name" binding:"-"`
	DestinationPort string    `gorm:"column:destination_port" json:"destination_port" binding:"-"`
	Protocol        string    `gorm:"column:protocol" json:"protocol" binding:"-"`
	IpVersion       string    `gorm:"column:ip_version" json:"ip_version" binding:"-"`
	Direction       string    `gorm:"column:direction" json:"direction" binding:"-"`
	Action          string    `gorm:"column:action" json:"action" binding:"-"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (t TInnerPolicy) TableName() string {
	return "t_inner_policy"
}

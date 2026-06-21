package db

import "time"

type Novel struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	UserID         uint64    `gorm:"type:bigint;index;not null" json:"user_id"`
	Title          string    `gorm:"type:varchar(200);not null" json:"title"`
	SourceFilename string    `gorm:"type:varchar(255)" json:"source_filename"`
	RawContent     string    `gorm:"type:longtext" json:"-"`
	ChapterCount   int       `gorm:"type:int;not null;default:0" json:"chapter_count"`
	Status         string    `gorm:"type:varchar(20);default:'active'" json:"status"`
	CreatedAt      time.Time `gorm:"type:datetime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"type:datetime" json:"updated_at"`
}

type NovelChapter struct {
	ID               uint64    `gorm:"primaryKey" json:"id"`
	NovelID          uint64    `gorm:"type:bigint;index;not null" json:"novel_id"`
	ChapterIndex     int       `gorm:"type:int;not null" json:"chapter_index"`
	Title            string    `gorm:"type:varchar(200)" json:"title"`
	Content          string    `gorm:"type:longtext" json:"content"`
	StoryboardStatus string    `gorm:"type:varchar(20);default:'none'" json:"storyboard_status"`
	CreatedAt        time.Time `gorm:"type:datetime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"type:datetime" json:"updated_at"`
}

type NovelShot struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	ChapterID  uint64    `gorm:"type:bigint;index;not null" json:"chapter_id"`
	ShotIndex  int       `gorm:"type:int;not null" json:"shot_index"`
	Scene      string    `gorm:"type:varchar(1000)" json:"scene"`
	Characters string    `gorm:"type:varchar(500)" json:"characters"`
	Prompt     string    `gorm:"type:varchar(2000)" json:"prompt"`
	Dialogue   string    `gorm:"type:varchar(1000)" json:"dialogue"`
	Camera     string    `gorm:"type:varchar(100)" json:"camera"`
	ImageURL   string    `gorm:"type:varchar(500)" json:"image_url"`
	VideoURL   string    `gorm:"type:varchar(500)" json:"video_url"`
	CreatedAt  time.Time `gorm:"type:datetime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"type:datetime" json:"updated_at"`
}

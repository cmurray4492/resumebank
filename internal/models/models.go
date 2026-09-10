// Package models defines the domain structs shared across the repo and web layers.
package models

import "time"

type Role string

const (
	RoleCandidate Role = "candidate"
	RoleEmployer  Role = "employer"
	RoleAdmin     Role = "admin"
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
}

type Candidate struct {
	ID               int64
	UserID           int64
	Slug             string
	Name             string
	Title            string
	City             string
	State            string
	Zipcode          string
	Email            string
	Phone            string
	LinkedInURL      string
	Skills           string
	Summary          string
	ResumeHTML       string
	ResumeText       string
	PhotoPath        string
	PhotoContentType string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Employer struct {
	ID              int64
	UserID          int64
	Slug            string
	CompanyName     string
	Industry        string
	City            string
	State           string
	Zipcode         string
	Phone           string
	EmailAddress    string
	Website         string
	Description     string
	Locations       string
	LogoPath        string
	LogoContentType string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Job struct {
	ID              int64
	EmployerID      int64
	Slug            string
	Title           string
	Location        string
	JobNumber       string
	SalaryMin       *int
	SalaryMax       *int
	DescriptionHTML string
	DescriptionText string
	ApplyMethod     string
	ApplyValue      string
	DatePosted      time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type FileKind string

const (
	FileKindResumePDF  FileKind = "resume_pdf"
	FileKindAdditional FileKind = "additional"
)

type CandidateFile struct {
	ID               int64
	CandidateID      int64
	Kind             FileKind
	OriginalFilename string
	StoredPath       string
	ContentType      string
	SizeBytes        int64
	Position         int16
	CreatedAt        time.Time
}

type JobVote struct {
	CandidateID int64
	JobID       int64
	Vote        int16 // +1 or -1
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Message struct {
	ID          int64
	SenderID    int64
	RecipientID int64
	Body        string
	CreatedAt   time.Time
	ReadAt      *time.Time
}

type BlogPost struct {
	ID          int64
	Slug        string
	Title       string
	BodyHTML    string
	BodyText    string
	AuthorName  string
	Published   bool
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

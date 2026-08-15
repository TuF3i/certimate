package domain

const CollectionNameOAuth2Link = "oauth2_link"

// OAuth2Link 表示 OAuth2 身份与 PocketBase 超级管理员账户的关联。
type OAuth2Link struct {
	Meta
	Provider          string `db:"provider"          json:"provider"`
	SubjectId         string `db:"subjectId"         json:"subjectId"`
	SuperuserId       string `db:"superuserId"       json:"superuserId"`
	UserProfileEmail  string `db:"userProfileEmail"  json:"userProfileEmail,omitempty"`
	UserProfileName   string `db:"userProfileName"   json:"userProfileName,omitempty"`
	UserProfileAvatar string `db:"userProfileAvatar" json:"userProfileAvatar,omitempty"`
}

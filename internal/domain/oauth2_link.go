package domain

const CollectionNameOAuth2Link = "oauth2_link"

// OAuth2Link 表示 OAuth2 身份与 Certimate 账号（超级管理员或普通用户）的关联。
type OAuth2Link struct {
	Meta
	Provider          string `db:"provider"          json:"provider"`
	SubjectId         string `db:"subjectId"         json:"subjectId"`
	TargetCollection  string `db:"targetCollection"  json:"targetCollection"` // 关联目标集合："_superusers" 或 "users"
	SuperuserId       string `db:"superuserId"       json:"superuserId"`      // 关联账号记录 ID
	UserProfileEmail  string `db:"userProfileEmail"  json:"userProfileEmail,omitempty"`
	UserProfileName   string `db:"userProfileName"   json:"userProfileName,omitempty"`
	UserProfileAvatar string `db:"userProfileAvatar" json:"userProfileAvatar,omitempty"`
}

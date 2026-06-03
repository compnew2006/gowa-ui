package models

import (
	"time"

	"github.com/google/uuid"
)

type FacebookCommentStatus = string

const (
	FBCommentStatusOpen     FacebookCommentStatus = "open"
	FBCommentStatusReplied  FacebookCommentStatus = "replied"
	FBCommentStatusClosed   FacebookCommentStatus = "closed"
	FBCommentStatusArchived FacebookCommentStatus = "archived"
)

type FacebookCommentDirection = string

const (
	FBCommentDirectionIncoming FacebookCommentDirection = "incoming"
	FBCommentDirectionOutgoing FacebookCommentDirection = "outgoing"
)

type FacebookComment struct {
	BaseModel
	OrganizationID uuid.UUID                `gorm:"type:uuid;not null;index;uniqueIndex:idx_fb_comment_external_org" json:"organization_id"`
	AccountID      uuid.UUID                `gorm:"type:uuid;not null;index" json:"account_id"`
	PageID         string                   `gorm:"size:255;not null;index:idx_fb_comment_page_external,priority:1" json:"page_id"`
	PageName       string                   `gorm:"size:255" json:"page_name"`
	PostID         string                   `gorm:"size:255;not null;index" json:"post_id"`
	PostPermalink  string                   `gorm:"type:text" json:"post_permalink"`
	PostMessage    string                   `gorm:"type:text" json:"post_message"`
	ExternalID     string                   `gorm:"size:255;not null;uniqueIndex:idx_fb_comment_external_org" json:"external_id"`
	ParentID       string                   `gorm:"size:255;index" json:"parent_id"`
	FromID         string                   `gorm:"size:255;index" json:"from_id"`
	FromName       string                   `gorm:"size:255" json:"from_name"`
	Message        string                   `gorm:"type:text" json:"message"`
	Permalink      string                   `gorm:"type:text" json:"permalink"`
	Status         FacebookCommentStatus    `gorm:"type:varchar(20);default:'open';index" json:"status"`
	Direction      FacebookCommentDirection `gorm:"type:varchar(20);default:'incoming';index" json:"direction"`
	CommentedAt    time.Time                `gorm:"index" json:"commented_at"`
	LastSyncedAt   *time.Time               `gorm:"index" json:"last_synced_at,omitempty"`
	LastRepliedAt  *time.Time               `gorm:"index" json:"last_replied_at,omitempty"`
	AutoRepliedAt  *time.Time               `gorm:"index" json:"auto_replied_at,omitempty"`
	Metadata       JSONB                    `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	Organization *Organization          `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Account      *FacebookAccount       `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	Replies      []FacebookCommentReply `gorm:"foreignKey:CommentID" json:"replies,omitempty"`
}

func (FacebookComment) TableName() string {
	return "facebook_comments"
}

type FacebookCommentReply struct {
	BaseModel
	OrganizationID      uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	CommentID           uuid.UUID `gorm:"type:uuid;not null;index" json:"comment_id"`
	AccountID           uuid.UUID `gorm:"type:uuid;not null;index" json:"account_id"`
	PageID              string    `gorm:"size:255;not null;index" json:"page_id"`
	UserID              uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	ReplyText           string    `gorm:"type:text" json:"reply_text"`
	PrivateMessageText  string    `gorm:"type:text" json:"private_message_text"`
	GraphCommentReplyID string    `gorm:"size:255" json:"graph_comment_reply_id"`
	GraphPrivateReplyID string    `gorm:"size:255" json:"graph_private_reply_id"`
	Status              string    `gorm:"size:40;default:'sent';index" json:"status"`
	ErrorMessage        string    `gorm:"type:text" json:"error_message,omitempty"`
	IsAuto              bool      `gorm:"default:false;index" json:"is_auto"`
	Metadata            JSONB     `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	Organization *Organization    `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Comment      *FacebookComment `gorm:"foreignKey:CommentID" json:"comment,omitempty"`
	Account      *FacebookAccount `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	User         *User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (FacebookCommentReply) TableName() string {
	return "facebook_comment_replies"
}

type FacebookCommentSettings struct {
	BaseModel
	OrganizationID             uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"organization_id"`
	Enabled                    bool      `gorm:"default:true" json:"enabled"`
	SyncEnabled                bool      `gorm:"default:true" json:"sync_enabled"`
	AutoReplyEnabled           bool      `gorm:"default:false" json:"auto_reply_enabled"`
	AutoCommentReplyEnabled    bool      `gorm:"default:true" json:"auto_comment_reply_enabled"`
	AutoPrivateReplyEnabled    bool      `gorm:"default:true" json:"auto_private_reply_enabled"`
	AutoCommentReplyText       string    `gorm:"type:text;default:'تم الرد خاص'" json:"auto_comment_reply_text"`
	AutoPrivateMessageText     string    `gorm:"type:text;default:'اهلا كيف اقدر اساعدك'" json:"auto_private_message_text"`
	OnlyAutoReplyUnanswered    bool      `gorm:"default:true" json:"only_auto_reply_unanswered"`
	IgnorePageAdminComments    bool      `gorm:"default:true" json:"ignore_page_admin_comments"`
	DefaultSyncPostLimit       int       `gorm:"default:25" json:"default_sync_post_limit"`
	DefaultSyncCommentsPerPost int       `gorm:"default:50" json:"default_sync_comments_per_post"`
	Metadata                   JSONB     `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

func (FacebookCommentSettings) TableName() string {
	return "facebook_comment_settings"
}

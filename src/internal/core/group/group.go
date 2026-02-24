// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package group manages scanlation groups and their memberships.

It handles the lifecycle of contributor organizations, from discovery and following
to internal role management and verification status.

# Core Responsibility

  - Organization: Defines the [Group] entity and its metadata.
  - Membership: Manages [Member] associations and hierarchical roles.
  - Verification: Tracks official publisher badges and moderation status.

This package provides the organizational context for content uploads in the core domain.
*/
package group

import "time"

// # Group Enums

// Role defines the authority level of a member within a group.
type Role string

const (
	RoleLeader    Role = "leader"
	RoleModerator Role = "moderator"
	RoleMember    Role = "member"
)

// # Core Entities

// Group represents an organization that translates and uploads content.
type Group struct {
	ID                  string     `json:"id"` // UUIDv7
	Name                string     `json:"name"`
	Slug                string     `json:"slug"`
	Description         *string    `json:"description,omitempty"`
	Website             *string    `json:"website,omitempty"`
	Discord             *string    `json:"discord,omitempty"`
	Twitter             *string    `json:"twitter,omitempty"`
	Patreon             *string    `json:"patreon,omitempty"`
	Youtube             *string    `json:"youtube,omitempty"`
	MangaUpdates        *string    `json:"manga_updates,omitempty"`
	IsOfficialPublisher bool       `json:"is_official_publisher"`
	IsActive            bool       `json:"is_active"`
	IsFocused           bool       `json:"is_focused"`
	VerifiedAt          *time.Time `json:"verified_at,omitempty"`
	MemberCount         int        `json:"member_count"`
	FollowCount         int        `json:"follow_count"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `json:"-"`
}

// Member represents a user's affiliation and role within a specific group.
type Member struct {
	GroupID     string    `json:"group_id"`
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`               // Denormalized for detail views
	DisplayName *string   `json:"display_name,omitempty"` // Denormalized for detail views
	AvatarURL   *string   `json:"avatar_url,omitempty"`   // Denormalized for detail views
	Role        Role      `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
}

// # Search & Filtering

// Filter holds parameters for searching and listing groups.
type Filter struct {
	Query               string `json:"q"`
	IsOfficialPublisher *bool  `json:"is_official_publisher"`
	IsFocused           *bool  `json:"is_focused"`
	Sort                string `json:"sort"` // name, followcount, createdat
}

// # Field Identifiers

const (
	FieldName         = "name"
	FieldDescription  = "description"
	FieldWebsite      = "website"
	FieldDiscord      = "discord"
	FieldTwitter      = "twitter"
	FieldPatreon      = "patreon"
	FieldYoutube      = "youtube"
	FieldMangaUpdates = "manga_updates"
	FieldSlug         = "slug"
	FieldItems        = "items"
	FieldTotal        = "total"
	FieldMessage      = "message"
)

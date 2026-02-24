// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package sec

// # User Roles

// UserRole represents the authorization level granted to an account.
type UserRole string

const (
	// Unrestricted system access
	RoleAdmin UserRole = "admin"

	// Can manage community content and moderate comments/users
	RoleModerator UserRole = "moderator"

	// Can upload and manage their own comic publications
	RoleAuthor UserRole = "author"

	// Default role for standard registered users
	RoleMember UserRole = "member"
)

// # Role Hierarchy

// AtLeast checks if the current role meets or exceeds the required target role.
func (r UserRole) AtLeast(target UserRole) bool {
	return r.level() >= target.level()
}

// level maps a role to a numeric hierarchy level for comparison logic.
func (r UserRole) level() int {

	// Linear scale (10-40) allows for future intermediate roles
	switch r {
	case RoleAdmin:
		return 40
	case RoleModerator:
		return 30
	case RoleAuthor:
		return 20
	case RoleMember:
		return 10
	default:
		return 0
	}
}

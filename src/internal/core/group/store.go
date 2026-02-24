// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package group

import "context"

// # Group Data Access

// Repository defines the data access contract for groups and memberships.
type Repository interface {

	/*
		List returns a filtered, paginated slice of groups and the total count.

		Parameters:
		  - context: context.Context
		  - filter: Filter (Search query, official status, etc.)
		  - limit: int
		  - offset: int

		Returns:
		  - []*Group: Slice of matching groups
		  - int: Total record count
		  - error: Database retrieval failures
	*/
	List(context context.Context, filter Filter, limit, offset int) ([]*Group, int, error)

	/*
		FindByID retrieves a group by its UUID.

		Parameters:
		  - context: context.Context
		  - id: string (UUIDv7)

		Returns:
		  - *Group: Hydrated entity
		  - error: ErrNotFound if missing or inactive
	*/
	FindByID(context context.Context, id string) (*Group, error)

	/*
		FindBySlug retrieves a group by its human-readable identifier.

		Parameters:
		  - context: context.Context
		  - slug: string

		Returns:
		  - *Group: Hydrated entity
		  - error: ErrNotFound if missing
	*/
	FindBySlug(context context.Context, slug string) (*Group, error)

	/*
		Create persists a new group to the store.

		Parameters:
		  - context: context.Context
		  - group: *Group

		Returns:
		  - error: Persistence failures
	*/
	Create(context context.Context, group *Group) error

	/*
		Update modifies an existing group's metadata.

		Parameters:
		  - context: context.Context
		  - group: *Group

		Returns:
		  - error: Persistence failures
	*/
	Update(context context.Context, group *Group) error

	/*
		SoftDelete marks a group as deleted.

		Parameters:
		  - context: context.Context
		  - id: string

		Returns:
		  - error: Persistence failures
	*/
	SoftDelete(context context.Context, id string) error

	// # Membership Management

	/*
		ListMembers returns all users affiliated with a group.

		Parameters:
		  - context: context.Context
		  - groupID: string

		Returns:
		  - []*Member: List of affiliated users
		  - error: Retrieval failures
	*/
	ListMembers(context context.Context, groupID string) ([]*Member, error)

	/*
		AddMember links a user to a group with a specified role.

		Parameters:
		  - context: context.Context
		  - member: *Member

		Returns:
		  - error: Persistence failures
	*/
	AddMember(context context.Context, member *Member) error

	/*
		UpdateMemberRole changes a user's authority level within a group.

		Parameters:
		  - context: context.Context
		  - groupID: string
		  - userID: string
		  - role: Role

		Returns:
		  - error: Persistence failures
	*/
	UpdateMemberRole(context context.Context, groupID, userID string, role Role) error

	/*
		RemoveMember terminates a user's affiliation with a group.

		Parameters:
		  - context: context.Context
		  - groupID: string
		  - userID: string

		Returns:
		  - error: Persistence failures
	*/
	RemoveMember(context context.Context, groupID, userID string) error

	// # Social & Following

	/*
		Follow increments follow count and creates a relationship.

		Parameters:
		  - context: context.Context
		  - groupID: string
		  - userID: string

		Returns:
		  - error: Persistence failures
	*/
	Follow(context context.Context, groupID, userID string) error

	/*
		Unfollow decrements follow count and removes the relationship.

		Parameters:
		  - context: context.Context
		  - groupID: string
		  - userID: string

		Returns:
		  - error: Persistence failures
	*/
	Unfollow(context context.Context, groupID, userID string) error
}

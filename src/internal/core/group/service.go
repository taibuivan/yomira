// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package group

import (
	"context"
	"log/slog"

	"github.com/taibuivan/yomira/internal/platform/validate"
	"github.com/taibuivan/yomira/pkg/slug"
	"github.com/taibuivan/yomira/pkg/uuid"
)

// # Service Layer

// Service orchestrates business rules for scanlation groups and memberships.
type Service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService constructs a new group [Service].
func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// # Group Management

/*
ListGroups retrieves a paginated and filtered list of scanlation groups.

Parameters:
  - context: context.Context
  - filter: Filter
  - limit, offset: int

Returns:
  - []*Group: List of groups
  - int: Total matching count
  - error: Retrieval errors
*/
func (service *Service) ListGroups(context context.Context, filter Filter, limit, offset int) ([]*Group, int, error) {
	return service.repo.List(context, filter, limit, offset)
}

/*
GetGroup retrieves a group by its UUID or Slug identifier.

Parameters:
  - context: context.Context
  - identifier: string

Returns:
  - *Group: Hydrated group entity
  - error: ErrNotFound if missing
*/
func (service *Service) GetGroup(context context.Context, identifier string) (*Group, error) {

	// Discriminator: ID vs Slug
	// UUIDs have a fixed length of 36 characters in standard hyphenated format.
	if len(identifier) == 36 {
		return service.repo.FindByID(context, identifier)
	}

	return service.repo.FindBySlug(context, identifier)
}

/*
CreateGroup initialises a new organization and assigns the creator as leader.

Parameters:
  - context: context.Context
  - group: *Group
  - creatorID: string (The user creating the group)

Returns:
  - error: Validation or persistence failures
*/
func (service *Service) CreateGroup(context context.Context, group *Group, creatorID string) error {
	validator := &validate.Validator{}
	validator.Required(FieldName, group.Name).MaxLen(FieldName, group.Name, 200)

	if group.Website != nil {
		validator.URL(FieldWebsite, *group.Website)
	}

	if err := validator.Err(); err != nil {
		return err
	}

	group.ID = uuid.New()
	group.Slug = slug.From(group.Name)
	group.IsActive = true

	if err := service.repo.Create(context, group); err != nil {
		return err
	}

	if err := service.repo.AddMember(context, &Member{
		GroupID: group.ID,
		UserID:  creatorID,
		Role:    RoleLeader,
	}); err != nil {
		return err
	}

	service.logger.Info("group_created",
		slog.String("group_id", group.ID),
		slog.String("creator_id", creatorID),
	)

	return nil
}

/*
UpdateGroup modifies the metadata of an existing group.

Parameters:
  - context: context.Context
  - group: *Group

Returns:
  - error: Validation or persistence failures
*/
func (service *Service) UpdateGroup(context context.Context, group *Group) error {
	validator := &validate.Validator{}
	if group.Name != "" {
		validator.MaxLen("name", group.Name, 200)
	}

	if group.Website != nil {
		validator.URL("website", *group.Website)
	}

	if err := validator.Err(); err != nil {
		return err
	}

	if err := service.repo.Update(context, group); err != nil {
		return err
	}

	service.logger.Info("group_updated", slog.String("group_id", group.ID))

	return nil
}

// # Membership Controls

/*
ListMembers returns the roster for a specific scanlation group.

Parameters:
  - context: context.Context
  - groupID: string

Returns:
  - []*Member: List of affiliated users
  - error: Retrieval failures
*/
func (service *Service) ListMembers(context context.Context, groupID string) ([]*Member, error) {
	return service.repo.ListMembers(context, groupID)
}

/*
AddMember invites or adds a new user to the group roster.

Parameters:
  - context: context.Context
  - m: *Member

Returns:
  - error: Verification or storage failures
*/
func (service *Service) AddMember(context context.Context, member *Member) error {
	// Verification logic (isactive user, etc) would go here
	return service.repo.AddMember(context, member)
}

/*
RemoveMember removes an affiliation between a user and a group.

Parameters:
  - context: context.Context
  - groupID: string (UUID)
  - userID: string (UUID)

Returns:
  - error: Storage failures
*/
func (service *Service) RemoveMember(context context.Context, groupID, userID string) error {
	return service.repo.RemoveMember(context, groupID, userID)
}

/*
FollowGroup records a user's interest in a scanlation group.

Parameters:
  - context: context.Context
  - groupID: string (UUID)
  - userID: string (UUID)

Returns:
  - error: Persistence failures
*/
func (service *Service) FollowGroup(context context.Context, groupID, userID string) error {
	if err := service.repo.Follow(context, groupID, userID); err != nil {
		return err
	}

	service.logger.Info("user_followed_group",
		slog.String("group_id", groupID),
		slog.String("user_id", userID),
	)

	return nil
}

/*
UnfollowGroup removes a group from a user's feed.

Parameters:
  - context: context.Context
  - groupID: string (UUID)
  - userID: string (UUID)

Returns:
  - error: Persistence failures
*/
func (service *Service) UnfollowGroup(context context.Context, groupID, userID string) error {
	if err := service.repo.Unfollow(context, groupID, userID); err != nil {
		return err
	}

	service.logger.Info("user_unfollowed_group",
		slog.String("group_id", groupID),
		slog.String("user_id", userID),
	)

	return nil
}

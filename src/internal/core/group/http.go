/*
Package group provides the HTTP interface for scanlation group management.

It exposes endpoints for group discovery, membership handling, and social interactions.

# Routing Strategy

  - Public (v1): Listing and detail views (GET /groups).
  - Authenticated: Member-specific actions (POST /groups, POST /groups/{id}/follow).
  - Restricted: Administrative or group-leader actions (PATCH, DELETE members).

The handler translates between the REST layer and the [Service] domain.
*/
package group

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/taibuivan/yomira/internal/platform/constants"
	requestutil "github.com/taibuivan/yomira/internal/platform/request"
	"github.com/taibuivan/yomira/internal/platform/respond"
	"github.com/taibuivan/yomira/internal/platform/validate"
	"github.com/taibuivan/yomira/pkg/pagination"
)

// # Handler Implementation

// Handler implements the HTTP layer for scanlation group operations.
type Handler struct {
	service *Service
}

// NewHandler constructs a new group [Handler].
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns a [chi.Router] configured with group-related endpoints.
func (handler *Handler) Routes() chi.Router {
	router := chi.NewRouter()

	// ## Public Discovery
	router.Get("/", handler.listGroups)
	router.Get("/{identifier}", handler.getGroup)
	router.Get("/{id}/members", handler.listMembers)

	// ## Social & Membership (Auth Required)
	// Note: Authentication middleware should be wrapped when mounting this router in main.go
	router.Post("/", handler.createGroup)
	router.Post("/{id}/follow", handler.followGroup)
	router.Delete("/{id}/follow", handler.unfollowGroup)

	// ## Administrative (Protected per-endpoint)
	router.Route("/{id}", func(subRouter chi.Router) {
		subRouter.Patch("/", handler.updateGroup)
		subRouter.Route("/members", func(members chi.Router) {
			members.Post("/", handler.addMember)
			members.Delete("/{userID}", handler.removeMember)
		})
	})

	return router
}

// # Group Endpoints

/*
GET /api/v1/groups.

Description: Retrieves a paginated list of scanlation groups.
Supports searching by name and filtering by verification status.

Request:
  - q: string (Full-text search)
  - isofficial: bool (Official publishers only)
  - limit: int
  - page: int

Response:
  - 200: []Group: Paginated list
*/
func (handler *Handler) listGroups(writer http.ResponseWriter, request *http.Request) {
	paginationParams := pagination.FromRequest(request)
	queryParams := request.URL.Query()

	filter := Filter{
		Query: queryParams.Get("q"),
	}

	if isOfficial := queryParams.Get("isofficial"); isOfficial != "" {
		value := isOfficial == "true"
		filter.IsOfficialPublisher = &value
	}

	groups, total, err := handler.service.ListGroups(request.Context(), filter, paginationParams.Limit, paginationParams.Offset())
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Paginated(writer, groups, pagination.NewMeta(paginationParams.Page, paginationParams.Limit, total))
}

/*
GET /api/v1/groups/{identifier}.

Description: Retrieves full details of a group using its UUID or unique title slug.

Request:
  - identifier: string (UUID or Slug)

Response:
  - 200: Group: Success
  - 400: 400: ErrInvalidID: Invalid identifier format
  - 404: 404: ErrNotFound: Group not found
*/
func (handler *Handler) getGroup(writer http.ResponseWriter, request *http.Request) {
	identifier := requestutil.Param(request, "identifier")

	group, err := handler.service.GetGroup(request.Context(), identifier)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, group)
}

/*
POST /api/v1/groups.

Description: Registers a new scanlation group.
Slugs are auto-generated from the group name.

Request (Body):
  - Group JSON object

Response:
  - 201: Group: Created object
  - 400: 400: ErrInvalidJSON/Validation: Invalid input data
  - 401: 401: ErrUnauthorized: Authentication required
*/
func (handler *Handler) createGroup(writer http.ResponseWriter, request *http.Request) {
	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	var input Group

	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	v := &validate.Validator{}
	v.Required("name", input.Name).MaxLen("name", input.Name, 200)
	if input.Website != nil && *input.Website != "" {
		v.URL("website", *input.Website)
	}

	if err := v.Err(); err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.service.CreateGroup(request.Context(), &input, userID); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Created(writer, input)
}

/*
PATCH /api/v1/groups/{id}.

Description: Updates mutable group metadata like description, website links, or name.

Request:
  - id: string (Target UUID)
  - body: Group Partial (JSON)

Response:
  - 200: Group: Updated entity
  - 400: 400: ErrInvalidJSON/Validation: Invalid input data
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Group not found
*/
func (handler *Handler) updateGroup(writer http.ResponseWriter, request *http.Request) {
	groupID := requestutil.ID(request, "id")

	var input Group
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	v := &validate.Validator{}
	if input.Name != "" {
		v.MaxLen("name", input.Name, 200)
	}
	if input.Website != nil && *input.Website != "" {
		v.URL("website", *input.Website)
	}

	if err := v.Err(); err != nil {
		respond.Error(writer, request, err)
		return
	}

	input.ID = groupID

	if err := handler.service.UpdateGroup(request.Context(), &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, input)
}

// # Membership & Social Endpoints

/*
GET /api/v1/groups/{id}/members.

Description: Lists all users and their respective roles within the group roster.

Request:
  - id: string (Group UUID)

Response:
  - 200: []Member: Success
  - 404: 404: ErrNotFound: Group not found
*/
func (handler *Handler) listMembers(writer http.ResponseWriter, request *http.Request) {
	groupID := requestutil.ID(request, "id")

	members, err := handler.service.ListMembers(request.Context(), groupID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, members)
}

/*
POST /api/v1/groups/{id}/follow.

Description: Follows a group to receive updates in the user feed.

Request:
  - id: string (Group UUID)

Response:
  - 201: Message: Success
  - 401: 401: ErrUnauthorized: Authentication required
  - 404: 404: ErrNotFound: Group not found
*/
func (handler *Handler) followGroup(writer http.ResponseWriter, request *http.Request) {
	groupID := requestutil.ID(request, "id")

	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.service.FollowGroup(request.Context(), groupID, userID); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Created(writer, map[string]string{constants.FieldMessage: "Following group"})
}

/*
DELETE /api/v1/groups/{id}/follow.

Description: Unfollows a group to stop receiving updates.

Request:
  - id: string (Group UUID)

Response:
  - 204: No Content: Success
  - 401: 401: ErrUnauthorized: Authentication required
  - 404: 404: ErrNotFound: Subscription not found
*/
func (handler *Handler) unfollowGroup(writer http.ResponseWriter, request *http.Request) {
	groupID := requestutil.ID(request, "id")

	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.service.UnfollowGroup(request.Context(), groupID, userID); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.NoContent(writer)
}

/*
POST /api/v1/groups/{id}/members.

Description: Invites or adds a new user to the group roster.

Request (Body):
  - { "userid": "string", "role": "string" }

Response:
  - 201: Member: Created affiliation
  - 400: 400: ErrInvalidJSON: Invalid payload
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Leader only
  - 404: 404: ErrNotFound: Group or User not found
*/
func (handler *Handler) addMember(writer http.ResponseWriter, request *http.Request) {
	groupID := requestutil.ID(request, "id")

	var input Member
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}
	input.GroupID = groupID

	if err := handler.service.AddMember(request.Context(), &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Created(writer, input)
}

/*
DELETE /api/v1/groups/{id}/members/{userID}.

Description: Removes a member's affiliation with the group.

Request:
  - id: string (Group UUID)
  - userID: string (User UUID)

Response:
  - 204: No Content: Success
  - 401: 401: ErrUnauthorized: Authentication required
  - 403: 403: ErrForbidden: Insufficient permissions
  - 404: 404: ErrNotFound: Member not found
*/
func (handler *Handler) removeMember(writer http.ResponseWriter, request *http.Request) {
	groupID := requestutil.ID(request, "id")
	userID := requestutil.ID(request, "userID")

	if err := handler.service.RemoveMember(request.Context(), groupID, userID); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.NoContent(writer)
}
